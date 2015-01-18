#!/bin/bash

cp ./leaps /usr/sbin/leaps

if [ ! -d /etc/leaps ]; then
	mkdir -p /etc/leaps
	cp -R ./static /etc/leaps/www
fi

if [ ! -f /etc/leaps/config.yaml ]; then
	echo "Installing fresh config file at /etc/leaps/config.yaml ..."
	./leaps --print-yaml > /etc/leaps/config.yaml
fi

if [[ "$1" == "--daemon" ]]; then
	echo "Setting up init.d script and leaps user ..."
	id -u leaps &>/dev/null || (useradd -s /bin/bash leaps && echo leaps:leaps | chpasswd)

	cp ./config/init.d/leaps /etc/init.d/leaps
	chmod 755 /etc/init.d/leaps
	chown root:root /etc/init.d/leaps

	update-rc.d leaps start 30 2 3 4 5 . stop 30 0 1 6 .
fi
