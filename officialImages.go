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
*/

package main

import (
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"regexp"
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

func main() {
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

	cmd := exec.Command("git",
		"clone",
		"https://github.com/docker-library/official-images",
	)
	if err := cmd.Run(); err != nil {
		fmt.Printf("git clone went wrong : %s\n", err)
	}

	// get files in the
	//files, err := ioutil.ReadDir(fmt.Sprintf("%s/official-images\library", in.gitpath))
	imagesPath := fmt.Sprintf("%s/official-images/library", in.gitpath)
	os.Chdir(imagesPath)
	files, err := ioutil.ReadDir(imagesPath)
	if err != nil {
		fmt.Printf("Error in readdir : %s\n.", err)
	}

	// create a map with to store all images
	images := make(map[string]img)
	images["scratch:latest"] = img{"scratch", "latest", "", make([]string, 30), ""}

	gitArchives := make(map[string]string)

	// get all the image names tags and paths
	//r, _ := regexp.Compile("(\\S+):\\s+(git://github.com/.+)@\\S+\\s?(\\S*)")
	r, _ := regexp.Compile("(\\S+):\\s+(git://github.com/.+)@\\S+\\s?(\\S*)\\n")
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
			fmt.Printf("\t%s\n", my_matches[ar][1]) // tag
			//fmt.Printf("\t%s\n",my_matches[ar][2]) // github archive
			fmt.Printf("\t\t%s\n", my_matches[ar][3]) // dir in the archive containing docker file
			// make directory and clone archive.
			tag := my_matches[ar][1]
			gitArch := my_matches[ar][2]
			archPath := my_matches[ar][3]
			archivePath := fmt.Sprintf("%s/%s", in.dockerfilepath, f.Name())

			fullName := fmt.Sprintf("%s:%s", f.Name(), tag)
			DockerfilePath := fmt.Sprintf("%s/%s/Dockerfile", archivePath, archPath)

			gitArchives[gitArch] = archivePath
			images[fullName] = img{f.Name(), tag, "", make([]string, 30), DockerfilePath}

		}
		fmt.Printf("\n\n")

	}

	fmt.Printf("%v\n\n", images)
	//fmt.Printf("%v\n", gitArchives)

	// now start pulling the Dockerfile archives

	// parse the Dockerfiles to see where it's comming from
	for gitUrl, localPath := range gitArchives {
		//fmt.Printf("k: %s , v: %s\n", k, v)
		os.MkdirAll(localPath, 0777)
		filePath := fmt.Sprintf("%s/.git", localPath)
		var cmd *exec.Cmd
		if _, err := os.Stat(filePath); err == nil {
			os.Chdir(localPath)
			cmd = exec.Command("git",
			"pull",
		)
		}else {
			cmd = exec.Command("git",
			"clone",
			gitUrl,
			localPath,
		)
		}
		
		if err := cmd.Run(); err != nil {
			fmt.Printf("git clone went wrong : %s\n", err)
		}
	}

	os.Chdir(in.gitpath)
	os.RemoveAll(in.gitpath)
	//DelWinDir(in.gitpath)

}
