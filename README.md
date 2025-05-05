# goblin
A package manager in Go  

The program reads from a `sources.yaml`  

# Installation  
To install goblin, download it from releases.  
Or build it from source.  
```
git clone https://github.com/Alexandre1a/goblin.git
cd goblin
go mod tidy
go build
```

Add the build directory to your PATH environment variable.  

# Usage  
## Install a package  
To install a package, just run `goblin install <package>` in your terminal.  
## Uninstall a package  
To uninstall a package, just run `goblin remove <package>` in your terminal.  
## Update a package  
To update a package, just run `goblin update <package>` in your terminal.  
The `<package>` field in the update can be empty, to update all pacakages.  
# Features  
At the moment, it has basic features like downloading packages and installing them.  
Use `goblin sync` to sync packages from a lock file.  
This can be useful to "replicate" a specific environment.  
Use `goblin` without any arguments to see the help message.  

# Limitations  
No dependency managment at the moment.  
Only single binaries are supported, I may add support for achives later.  
