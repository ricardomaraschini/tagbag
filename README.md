<p align="center">
    <img src="banner.png" alt="Banner">
</p>

_This README was created by ChatGPT_

Embark on a journey with TAGBAG, the mystical tool that bundles multiple container
images into a single, nifty `tgz` file. It's like a magic trick for your images,
where the duplicated blobs get vanished - Poof! - ensuring each one is as unique as
a snowflake.

## How to Use This Wizardry

### Conjuring Images from the Ether

Summon a plethora of images and securely tuck them into `images.tgz`:

```
$ tagbag pull                           \
        -i alpine:latest                \
        -i myrepo/myimage:latest        \
        -d images.tgz
```

During this mystical pull, TAGBAG cleverly sifts through the layers, ensuring duplicates
are as rare as unicorns.

For authentication, you have two potions to choose from: Cast a spell with docker login
to access your desired registry realm, or craft an arcane authentication scroll named
`auth.json`:

```json
{
    "auths": {
        "https://index.docker.io/v1/": {
            "auth": "bXl1c2VyOm15cGFzc3dvcmQ="
        }
    }
}
```

Invoke images with your scroll like so:

```
$ tagbag pull                           \
        --authfile auth.json            \
        -i alpine:latest                \
        -i myrepo/myimage:latest        \
        -d images.tgz
```

### Releasing Images Back into the Wild

Ready to set your images free? Dispatch them to a new domain with this incantation:

```
$ tagbag push                   \
        --authfile auth.json    \
        -s images.tgz           \
        --destination docker.io/myaccount
```

Remember, if you've captured `alpine:latest` in your tgz spell, it'll be released into
the wilds of `docker.io/myaccount/alpine:latest`. All the captured images, regardless
of their origin, will find a new home in this singular mystical repository.


### Brewing Diffs Between Mystical Bundles

In the realm of TAGBAG, you have the power to unveil the subtle changes between different
versions of a given `tgz` bundle. First, ensure both versions of your mystical bundle are
safely harbored in your local sanctum. Then, chant the following incantation:


```
$ tagbag diff			\
	--source v1.0.0.tgz	\
	--target v2.0.0.tgz	\
	-d overlay.tgz
```

With this spell cast, the veil is lifted, revealing a diff – a magical overlay that
captures the essence of change. To bestow this newfound knowledge upon an existing
tgz, perform the following ritual:


```
$ tagbag push			\
	--source v1.0.0.tgz	\
	--overlay overlay.tgz	\
	-d docker.io/myaccount
```

This will turn your `v1.0.0.tgz` into a `v2.0.0.tgz` and then push to the remote
registry.
