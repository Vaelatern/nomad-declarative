# Orchestrated Jobs As Configuration

Nomad and k3s are awesome.

Unfortunately, there isn't really a paved path for operators to define their entire compute cluster jobs as a consistent set of programs that are running.

Sure you can always reconstruct your cluster state by re-applying the same files you applied before, but which files DID you apply before? That's what I'm hoping to help solve here.

## How To Think Of This Tool

Let's declare in a configuration file (or `.d` style configuration files within a directory) a set of Jobs.

Those Jobs can come from Packs.

Those Jobs can contain multiple nomad jobs (one per file), variables to apply to Nomad, policies, plain text to reference, and whatever else you can think of.

These Packs can come from any reachable origin. We use the [go-fsimpl](https://pkg.go.dev/github.com/hairyhenderson/go-fsimpl) library to ease selection of pack remotes.

Packs have names. Packs have origins. Packs can have a different name at the origin than we name them ourselves.

Jobs have names. Those have to be unique in the output. That uniqueness is not enforced in the configuration file.

If you have anything that needs to be templated based on your job name, just template it. It's fine.
