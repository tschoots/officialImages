package main 

import (
    "flag"
    "fmt"
    "os"
    "os/exec"
    "io"
)

type input struct {
	gitpath string
}

var in input

func init() {
	flag.StringVar(&in.gitpath, "p", "REQUIRED", "the path where a git archive can be cloned.")
}

const APP_VERSION = "0.1"

// The flag package provides a default help printer via -h switch
var versionFlag *bool = flag.Bool("v", false, "Print the version number.")

func IsDirEmpty(name string) (bool, error) {
	f, err := os.Open(name)
	if err != nil {
		return false, err
	}
	defer f.Close()
	
	_,err = f.Readdirnames(1)
	if err == io.EOF {
		return true , nil
	}
	return false, err  // either error or not empty
}



func main() {
    flag.Parse() // Scan the arguments list 

    if *versionFlag {
        fmt.Println("Version:", APP_VERSION)
    }
    
    if in.gitpath != "REQUIRED"  {
    	fmt.Println("git path : ", in.gitpath)
    }
    
    os.Chdir(in.gitpath)
    
    curdir, err := os.Getwd()
    if err != nil {
    	fmt.Printf("pwd err : %s\n", err)
    }
    
    // check if the directory is empty
    empty, err := IsDirEmpty(curdir)
    if ! empty {
    	fmt.Printf("dir %s is not empty.", curdir)
    	os.Exit(0)
    }else if err != nil {
    	fmt.Printf("error opening dir %s \nwith error : %s\n", curdir, err)
    	os.Exit(0)
    }
    
    fmt.Printf("current directory : %s\n", curdir)
    
    cmd := exec.Command( "git",
    	"clone",
    	"https://github.com/docker-library/official-images",
    )
    if err :=  cmd.Run(); err != nil {
    	fmt.Printf("git clone went wrong : %s\n", err)
    }
    
    // get files in the 
    
    
    
    os.RemoveAll(in.gitpath)
}

