# goblin
A package manager in Go

# Features
This package manager can download from a `source.yaml` file.

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
To install a package, just run `goblin install <package>` in your terminal.
To uninstall a package, just run `goblin remove <package>` in your terminal.
To update a package, just run `goblin update <package>` in your terminal.
The `<package>` field in the update can be empty, to update all pacakages.
At the moment, it has basic features like downloading packages and installing them.
Use `goblin sync` to sync packages from a lock file.
This can be useful to "replicate" a specific environment.
Use `goblin` without any arguments to see the help message.
Also no dependency managment.
