# cs

calculate checksums

It's a simpler version of shasum + md5sum, but only for md5, sha1, sha256, and
sha512.

## usage

    cs -a 256 < foo.txt
    cs foo.txt
    cs -a sha1 foo.txt foo.txt foo.txt
