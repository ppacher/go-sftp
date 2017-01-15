# `go-sftp`

An SFTP implementation for GoLang. 

Support for server and clients is planned, however, I'm still working on the
client/protocol part. 

To try `go-sftp` run the following command (you'll need OpenSSH installed):

```bash
$ go run cmd/sftp/main.go --server /usr/lib/openssh/sftp-server
```

It will most likely fail some where ....


Low-level packet definitions according to SFTP Version 3 are located within the
`sshfxp` sub-package.
