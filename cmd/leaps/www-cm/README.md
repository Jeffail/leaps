Leaps with CodeMirror and Vue.js
================================

This is an alternative web UI being built with [CodeMirror][0] and [Vue.js][1],
the reason for making this is to gain full support for unicode. CodeMirror
solves many problems that would otherwise require nasty work-arounds in ACE.

For example, in ACE it is currently possible to delete a single character of a
surrogate pair, and converting that change into a transform is impossible
without some extremely dirty behaviour.

The reason for bringing Vue.js into the mix is to make my life easier whilst I
try and jazz up the functionality of the page. There are lots of customization
options for both ACE and CodeMirror that I would like to expose, as well as
quality of life buttons/tabs etc.

## To Build

### CodeMirror

CodeMirror is vendored at `vendor/github.com/codemirror/codemirror`, but the
source needs building with nodejs. If you have node installed you can build by
running `npm install` in the codemirror directory, which will construct our
codemirror files that are symlinked into this directory.

### Vue.js

## Plans Moving Forward

I intend to swap to this new UI as soon as possible, which means compiling it
into the binary. I will, however, leave the previous UI available in this repo.

[0]: https://github.com/codemirror/codemirror
[1]: https://github.com/vuejs/vue
