#!/bin/bash
id -u leaps &>/dev/null || (useradd -s /bin/bash leaps && echo leaps:leaps | chpasswd)

cp ./leaps /usr/sbin/leaps

if [ ! -d /etc/leaps ]; then
	mkdir -p /etc/leaps
	cp -R ./static /etc/leaps/www
fi

if [ ! -f /etc/leaps/config.yaml ]; then
	echo "Installing fresh config file at /etc/leaps/config.yaml ..."
	./leaps --print-yaml > /etc/leaps/config.yaml
fi

# cp ./config/init.d/leaps /etc/init.d/leaps
# chmod 755 /etc/init.d/leaps
# chown root:root /etc/init.d/leaps

# sudo update-rc.d leaps defaults
# sudo update-rc.d leaps enable
