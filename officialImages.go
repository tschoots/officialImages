/*
	author : Ton Schoots

	this program is dependent on a docker context.
	It should be run in the context of a docker-machine.

	Status :

	   9 - 12 - 2015
	        Alle official images tags are parsed.

	   13 - 12 - 2015
	   		Everthing is pulled from git.
	   		But it should be in parellel to speed up things
	   		And parsing of dockerfiles stil has to tacke place
	   		And generation of .dot file

	    16 - 12 -2015
	    	inconsistencies in the docker env it looks like for example
	    	ubuntu-upstart:utopic --> "ubuntu:14.10"
	    	But "ubuntu:14.10" not in the archive, and not on hub.docker.com
	    	
	    19 - 12 -2015
	        Created a piece of code generating a map for test code.
	        .dot file generation is to big for Graphviz and for the stack so have to figure out something else
*/

package main

import (
	"bufio"
	//"builtin"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"time"
	"sort"
	"bytes"
	
)

type input struct {
	gitpath        string
	dockerfilepath string
}

var in input

func init() {
	flag.StringVar(&in.gitpath, "p", "REQUIRED", "the path where a git archive can be cloned.")
	flag.StringVar(&in.dockerfilepath, "d", "REQUIRED", "the path where the archives containing Dockerfiles should reside.")
}

const APP_VERSION = "0.1"

// The flag package provides a default help printer via -h switch
var versionFlag *bool = flag.Bool("v", false, "Print the version number.")

type img struct {
	Name           string
	Tag            string
	From           string
	Childs         []string
	DockerfilePath string
}



func IsDirEmpty(name string) (bool, error) {
	f, err := os.Open(name)
	if err != nil {
		return false, err
	}
	defer f.Close()

	_, err = f.Readdirnames(1)
	if err == io.EOF {
		return true, nil
	}
	return false, err // either error or not empty
}

func DelWinDir(path string) {
	cmd := exec.Command("cmd",
		"/C",
		"rmdir /S /Q",
		path,
	)
	fmt.Println(cmd.Args)
	if err := cmd.Run(); err != nil {
		fmt.Printf("rmdir on windows went wrong : %s\n", err)
	}
}

func GetFromReference(pathToDockerFile string) (string, bool) {
	file, err := os.Open(pathToDockerFile)
	if err != nil {
		fmt.Printf("Error opening file: %s\n", pathToDockerFile)
		//dockerfile not there so point to scratch
		return "scratch:latest", true
	}
	defer file.Close()

	fromr, _ := regexp.Compile(`(?i)^\s*FROM\s*(\S+)`)
	scanner := bufio.NewScanner(file)
	scanner.Split(bufio.ScanLines)
	for scanner.Scan() {
		fmt.Println(scanner.Text())
		my_match := fromr.FindStringSubmatch(scanner.Text())
		if len(my_match) > 0 {
			from_str := my_match[1]
			if !strings.Contains(from_str, ":") {
				from_str = fmt.Sprintf("%s:latest", from_str)
			}
			return from_str, true
		}
	}
	fmt.Printf("No from ref in file : %s\n", pathToDockerFile)
	return "", false
}

func PullGitArch(gitUrl string, localPath string) {
	os.MkdirAll(localPath, 0777)
	filePath := fmt.Sprintf("%s%c.git", localPath, os.PathSeparator)
	var cmd *exec.Cmd
	if _, err := os.Stat(filePath); err == nil {
		os.Chdir(localPath)
		cmd = exec.Command("git",
			"pull",
			"-q",
		)
		fmt.Printf("pull : %s\n\n", localPath)
	} else {
		cmd = exec.Command("git",
			"clone",
			gitUrl,
			localPath,
		)
		fmt.Printf("clone : \n%s\n", cmd.Args)
	}

	if err := cmd.Run(); err != nil {
		fmt.Printf("git clone/pull went wrong : %s\n%s\n%s\n\n", err, gitUrl, localPath)
	}
}

func PullDockerfileArchives(gitArchives map[string]string) {
	var wg sync.WaitGroup

	for gitUrl, localPath := range gitArchives {
		//fmt.Printf("k: %s , v: %s\n", k, v)
		url := gitUrl
		path := localPath
		wg.Add(1)

		go func() {
			defer wg.Done()
			PullGitArch(url, path)
		}()
	}
	wg.Wait()
}

func main() {
	start_time := time.Now()
	flag.Parse() // Scan the arguments list

	if *versionFlag {
		fmt.Println("Version:", APP_VERSION)
	}

	if in.gitpath != "REQUIRED" {
		fmt.Println("git path : ", in.gitpath)
	}

	os.MkdirAll(in.gitpath, 0777)
	os.MkdirAll(in.dockerfilepath, 0777)
	err := os.Chdir(in.gitpath)
	if err != nil {
		fmt.Printf("error chdir : %s\n", err)
		os.Exit(0)
	}

	curdir, err := os.Getwd()
	if err != nil {
		fmt.Printf("pwd err : %s\n", err)
	} else {
		fmt.Printf("curr dir : %s\n", curdir)
	}

	// check if the directory is empty
	empty, err := IsDirEmpty(curdir)
	if !empty {
		fmt.Printf("dir %s is not empty.", curdir)
		os.Exit(0)
	} else if err != nil {
		fmt.Printf("error opening dir %s \nwith error : %s\n", curdir, err)
		os.Exit(0)
	}

	fmt.Printf("current directory : %s\n", curdir)

	//	cmd := exec.Command("git",
	//		"clone",
	//		"https://github.com/docker-library/official-images",
	//	)
	//	if err := cmd.Run(); err != nil {
	//		fmt.Printf("git clone went wrong : %s\n", err)
	//	}

	PullGitArch("https://github.com/docker-library/official-images", in.gitpath)

	// get files in the
	//files, err := ioutil.ReadDir(fmt.Sprintf("%s/official-images\library", in.gitpath))
	imagesPath := fmt.Sprintf("%s%clibrary", in.gitpath, os.PathSeparator)
	os.Chdir(imagesPath)
	files, err := ioutil.ReadDir(imagesPath)
	if err != nil {
		fmt.Printf("Error in readdir : %s\n.", err)
	}

	// create a map with to store all images
	images := make(map[string]img)
	images["scratch:latest"] = img{"scratch", "latest", "", make([]string, 0), ""}

	gitArchives := make(map[string]string)

	// get all the image names tags and paths
	r, _ := regexp.Compile(`(\S+):\s+(git://github.com/\S+)@\S+[\t\f\v\x20]*([^\n\r]*)`)
	for _, f := range files {
		fmt.Println(f.Name())
		content, err := ioutil.ReadFile(f.Name())
		if err != nil {
			fmt.Printf("error opening file : %s \n", f.Name())
		}
		//fmt.Printf("%s\n\n", string(content))
		my_matches := r.FindAllStringSubmatch(string(content), -1)

		for ar := range my_matches {
			//fmt.Println(len(my_matches[ar]))
			fmt.Printf("total: %s\n", my_matches[ar][0])
			fmt.Printf("tag:\t%s\n", my_matches[ar][1]) // tag
			//fmt.Printf("\t%s\n",my_matches[ar][2]) // github archive
			//fmt.Printf("\t\t%s\n", my_matches[ar][3]) // dir in the archive containing docker file
			// make directory and clone archive.
			tag := my_matches[ar][1]
			gitArch := my_matches[ar][2]
			archPath := my_matches[ar][3]
			archivePath := fmt.Sprintf("%s%c%s", in.dockerfilepath, os.PathSeparator, f.Name())

			fullName := fmt.Sprintf("%s:%s", f.Name(), tag)
			//DockerfilePath := fmt.Sprintf("%s%c%s%cDockerfile", archivePath, os.PathSeparator, archPath, os.PathSeparator)
			var DockerfilePath string
			if len(archPath) > 0 {
				fmt.Println("dir")
				DockerfilePath = fmt.Sprintf("%s%c%s%cDockerfile", archivePath, os.PathSeparator, archPath, os.PathSeparator)
			} else {
				fmt.Println("no dir")
				DockerfilePath = fmt.Sprintf("%s%cDockerfile", archivePath, os.PathSeparator)
			}

			gitArchives[gitArch] = archivePath
			images[fullName] = img{f.Name(), tag, "", make([]string, 0), DockerfilePath}

		}
		fmt.Printf("\n\n")

	}

	//fmt.Printf("%v\n", gitArchives)

	// now start pulling the Dockerfile archives
	PullDockerfileArchives(gitArchives)

	// parse the Dockerfiles to see where it's comming from
	for k, image := range images {
		fmt.Printf("name : %s:%s\n%s\n\n", image.Name, image.Tag, image.DockerfilePath)
		if from_ref, ok := GetFromReference(image.DockerfilePath); ok {
			// check if the reference exists
			if val, ok := images[from_ref]; ok {
				// update the from reference in the current image
				tmp_img := img{image.Name, image.Tag, from_ref, image.Childs, image.DockerfilePath}
				images[k] = tmp_img
				// update the child array of the referenced image
				tmp_arr := append(val.Childs, k)
				tmp_ref_img := img{val.Name, val.Tag, val.From, tmp_arr, val.DockerfilePath}
				images[from_ref] = tmp_ref_img

			} else {
				fmt.Printf("ERROR Docker file : %s, has a from reference that doesn't exists\n", image.DockerfilePath)
			}
			//
		} else {
			fmt.Printf("ERROR Dockerfile %s doesn't exist or from reference couldn't be parsed.\n", image.DockerfilePath)
		}
	}

	//fmt.Printf("%v\n\n", images)
	for k, v := range images {
		fmt.Printf("%-35s --> %s\n", k, v.From)
		sort.Strings(v.Childs)
		fmt.Printf("Childs : %q\n", v.Childs)
		fmt.Printf("Dockerfile : %s\n", v.DockerfilePath)
	}
	
	// build the dot file
	var b bytes.Buffer
	b.WriteString("digraph images_graph {\r\n")
	//b.WriteString("\tprr -> trr\r\n")
	dot_ids := make(map[string]int)
	id := 1
	for k, _ := range images {
		b.WriteString(fmt.Sprintf("\t%d[label=\"%s\"]\r\n", id, k))
		dot_ids[k] = id
		id++
	}
	image_name := "scratch:latest"
	for _, child_name := range images[image_name].Childs {
		b.WriteString(fmt.Sprintf("\t%d -> %d\r\n", dot_ids[image_name], dot_ids[child_name]))
	}
	b.WriteString("}")
	ioutil.WriteFile(`c:\tmp\dot\images.dot`, b.Bytes(), 0777)

	os.Chdir(in.gitpath)
	os.RemoveAll(in.gitpath)
	DelWinDir(in.gitpath)
	
	for k, v := range images {
		var arr_str string 
		if len(v.Childs) > 0 {
			var b bytes.Buffer
			b.WriteString("[]string{")
			for _, v := range v.Childs {
				b.WriteString(fmt.Sprintf(" \"%s\", ", v))
			}
			b.WriteString("}")
			arr_str = b.String()
		}else{
			arr_str = "make([]string, 0)"
		}
		fmt.Printf("\"%s\":img{\"%s\", \"%s\", \"%s\",  %s , `%s`,},\n", k, v.Name, v.Tag, v.From, arr_str, v.DockerfilePath)
			
	}

	elapsed_time := time.Since(start_time)
	fmt.Printf("time : %s", elapsed_time)
	fmt.Println("the end")

}
