# Summary

The repo-archive program downloads a number of Git repos then pushes them to a S3 bucket of your choice.

# Usage

Configure:
```
# in $HOME/.repo_archive.toml
[aws]
s3_bucket = "repo-archive.foo.bucket"
s3_region = "us-west-2"

[[repo]]
ID = "repo_1"
url = "https://github.com/example/repo1.git"
access_id = "user_name"
access_key = "foobar"

[[repo]]
ID = "repo_2"
url = "git://bitbucket.org/example/org.git"
```

Run
```
go run main.go --config $HOME/.repo_archive.toml
```

Expected output
```
Using config file: /home/jcheng/.repo_archive.toml
2019/08/15 21:34:39 cloning sites
2019/08/15 21:34:40 tarball created at /tmp/repo_archive_repo_id_1447775133/repo_id_1.tgz
2019/08/15 21:34:40 uploading /tmp/repo_archive_repo_id_1447775133/repo_id_1.tgz
2019/08/15 21:34:41 uploaded to  https://<your bucket>/data/repo_id_1/repo_id_1-2019-08-15.tgz
...
```
