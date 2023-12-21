# TAGBAG

Allows for transportation of multipe container images in a single tgz
file while deduplicating blobs referred by more than one image.

# Usage

## Pulling images

Pulls multiple images and saves them in the `images.tgz` file:
```
$ tagbag pull				\
	-i alpine:latest		\
	-i myrepo/myimage:latest	\
	-d images.tgz
```

For authentication you can either use `docker login` to login to the
registry you want to pull from or create an authentication file and
then refer to it with `--authfile` flag. For example, taking this
authentication file (called `auth.json`):

```json
{
    "auths": {
        "https://index.docker.io/v1/": {
            "auth": "bXl1c2VyOm15cGFzc3dvcmQ="
        }
    }
}
```

You can pull images using it with:

```
$ tagbag pull				\
	--authfile auth.json		\
	-i alpine:latest		\
	-i myrepo/myimage:latest	\
	-d images.tgz
```


## Pushing images

You can push back the images to a different registry with:

```
$ tagbag push			\
	--authfile auth.json	\
	-s images.tgz		\
	--destination docker.io/myaccount
```

Bear in mind that if you pulled `alpine:latest` it will be pushed
to `docker.io/myaccount/alpine:latest`. All images pulled, regardless
of the source repository, are pushed to the same repository.
