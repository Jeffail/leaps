##leapclient

This is the JavaScript implementation of a leaps client. It holds an internal model for processing incoming and outgoing transforms, and tools for automatically binding this model around various web page UI elements (currently textarea is the only element supported, but various web text editors are planned to be supported).

To test:

```bash
sudo npm install nodeunit -g
nodeunit test_leapclient.js
```

STATUS: TESTING INCOMPLETE

TODO:

- Create more auto tests and more in depth user story tests.
- More text editor binders.
