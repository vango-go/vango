// Package config provides configuration parsing for Vango projects.
//
// The configuration is stored in vango.json at the project root.
// This package handles loading, saving, and validating configuration.
//
// # Configuration File Structure
//
//	{
//	  "dev": {
//	    "port": 3000,
//	    "host": "localhost",
//	    "openBrowser": true,
//	    "https": false,
//	    "proxy": {
//	      "/api/external": "https://api.example.com"
//	    }
//	  },
//	  "build": {
//	    "output": "dist",
//	    "minify": true,
//	    "sourceMaps": false
//	  },
//	  "tailwind": {
//	    "enabled": true,
//	    "config": "./tailwind.config.js"
//	  },
//	  "ui": {
//	    "version": "1.0.0",
//	    "registry": "https://vango.dev/registry.json",
//	    "installed": ["button", "card"]
//	  },
//	  "hooks": "./public/js/hooks.js"
//	}
//
// # Usage
//
//	cfg, err := config.Load(".")
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	fmt.Println("Port:", cfg.Dev.Port)
package config
