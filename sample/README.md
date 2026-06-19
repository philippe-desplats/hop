# hop demo workspace

A self-contained playground to try `hop` without touching your real index, and the source of the README demo GIF.

It indexes **only** a throwaway copy of `sample/projects` (placed in `~/hop-demo`) and keeps `hop`'s own index in a temporary directory, through `XDG_CONFIG_HOME` and `XDG_STATE_HOME`. Your real `~/.config/hop` and project index are never touched. Open a fresh shell afterwards to return to your normal setup.

## Try it live

From the repository root, with `hop` on your `PATH`:

```sh
source sample/demo.sh
p api        # jump straight to acme-api
p            # open the Hub, then type to filter, Tab for actions
p -          # jump back
```

## Regenerate the README GIF

```sh
brew install vhs        # needs ttyd and ffmpeg too
vhs sample/demo.tape    # writes docs/demo.gif
```

## The sample projects

```
work/acme-api          Go      git, today
work/web-monorepo      Node    git, a few days ago
work/toolbox           Make    plain folder (no git preview)
side/blog              Astro   git, two weeks ago, branch "trunk"
side/pixel-game        HTML    plain folder (no git preview)
experiments/llm-playground  Python   git, yesterday
```

The names are fictitious and the categories (`work`, `side`, `experiments`) exist to show how `hop` colors projects by their top-level folder.
