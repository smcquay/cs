# cs

concurrently calculate/verify checksums (cs)

It's a simpler version of shasum + md5sum, but concurrently and only with
support for md5, sha1, sha256, and sha512.

## usage

    # create checksums
    cs -a 256 < foo.txt
    cs foo.txt
    cs -a sha1 foo.txt foo.txt foo.txt > checksums.sha1

    # verify
    cat checksums.sha1 | cs -c 
    cs -c checksums.sha1

    # both
    cs $(find ~/src/mcquay.me | grep  '\.go$') | cs -c
