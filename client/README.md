Client
======

This is the JavaScript implementation of a leaps client. It holds an internal model for processing incoming and outgoing transforms, and tools for automatically binding this model around various web page UI elements (currently supports textarea, ACE editor, and CodeMirror).

To test:

```bash
sudo npm install -g nodeunit jshint
jshint ./*.js
nodeunit test_leapclient.js
```
