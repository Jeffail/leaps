Installing leaps
================

Leaps is just a binary without runtime dependencies, so it can be stored anywhere and run. This is a simple guide demonstrating the quick and easy way to set leaps up on any linux/osx machine, but the reality is that you can put it anywhere that you'd like.

The first step for installation was extracting the .tar.gz or cloning the repository, well done!

The next step is to place the repository/package somewhere memorable and out of the way, it's easier for you to keep the static and config files along with the binary, so I tend to copy the entire leaps folder to `/opt`. If you do this then `ls /opt/leaps` should give you:

```
bin  config  docs  js  scripts  static
```

Next you will want to write a config file, if you're working from a fresh folder then you can start yourself off by copying an example file:

```bash
cd /opt/leaps
cp ./config/leaps_share.yaml ./config.yaml
```

For more information about leaps configuration check the docs folder.

When leaps is run it will automatically find and use the `config.yaml` file in the leaps folder, regardless of where that folder is stored, because it searches from the location of the binary itself.

Now, if you add `/opt/leaps/bin` to your PATH environment variable then you can run it at any time. However, if you wish to run leaps as a background service then we have an example init.d or supervisor config to use.

## init.d

Theres an example init.d script to use, this script assumes you copied the leaps directory to `/opt`, so quickly edit the file before moving on. It also assumes that there is a user "leaps" on your machine to run the service as, you can also change this or create the user:

```bash
useradd -s /bin/bash leaps

cp ./scripts/init.d/leaps /etc/init.d/leaps
chmod 755 /etc/init.d/leaps
chown root:root /etc/init.d/leaps

update-rc.d leaps start 30 2 3 4 5 . stop 30 0 1 6 .
```

## supervisor

The example supervisor config for leaps is much simpler, it also assumes you want to run the service as a user "leaps" and that the leaps folder is stored in `/opt`, just edit these values in the config to match reality and then copy it over:

```bash
useradd -s /bin/bash leaps

cp ./scripts/supervisor/conf.d/leaps.conf /etc/supervisor/conf.d/leaps.conf
```
