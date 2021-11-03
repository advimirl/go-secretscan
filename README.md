### Info
**go-secretscan** scans your gitlab/bitbucket repositories and tries to find stored "secrets" there.

This software is inspired with sshgit, that scans your entire gitlab/bitbucket server without downloading any files to disk.

Despite it's opensource nature, some features are not included in this version (IM integration, rate-limiting, etc.)


### Building

Nothing special, just:
```shell
$ go get github.com/doublestraus/go-secretscan
$ go-secretscan -config /path/to/config.yaml -signature /path/to/signatures.yaml
```
Scanner will create a folder named `reports` that stores scan reports in Gitlab Secrets Detection format (more https://docs.gitlab.com/ee/user/application_security/sast/index.html#reports-json-format)

Optionally, you can provide arguments `dd-url, dd-token, dd-product` to setup Defect Dodjo integration. Scanner will create a Defect Dodjo engagement for every repository and will import a JSON report there.


### Configuring

**config.yaml**

Must contain next fields:
```yaml
access_tokens:
- token: "" # Auth token for server, you can use file://path/to/token.txt form 
  base_url: "URL_OF_SERVER" # ex: gitlab.site.com
  worker_type: "bitbucket" #bitbucker/gitlab
  
- token: ""
  base_url: ""
  worker_type: "gitlab"
  ...

blacklisted_strings: ["AKIAIOSFODNN7EXAMPLE", "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY", "username:password", "sshpass -p $SSH_PASS"]
blacklisted_extensions: [".exe", ".dll", ".resx", ".so", ".min.js", ".pak", ".tar.xz", ".rar", ".gzip", ".jpg", ".iso", ".jpeg", ".png", ".gif", ".bmp", ".tiff", ".tif", ".psd", ".xcf", ".zip", ".tar.gz", ".ttf", ".lock", ".a"]
blacklisted_paths: ["node_modules{sep}", "vendorjs{sep}", "public{sep}js", "templates{sep}", "vendor{sep}bundle", "acme.sh{sep}", "boost{sep}", "jre{sep}lib", "vendor{sep}cache", "{sep}test{sep}", "{sep}tests{sep}", "example{sep}", "examples{sep}", ".vs{sep}"] # use {sep} for the OS' path seperator (i.e. / or \)
blacklisted_entropy_extensions: [".pem", ".key", "id_rsa", ".asc", ".ovpn", ".sqlite", ".sqlite3", ".log"] # You can blacklist file extensions
blacklisted_filenames: ["angular.js", "public.key", "test"] # You can blacklist filename 
blacklisted_project_names: ["3rdparties", "Autotest"] # You can blacklist path or concrete project name (path form)


```

**signature.yaml**

File with signatures provided from us stored in config/signatures.yaml.
