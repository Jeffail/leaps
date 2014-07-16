#!/bin/bash
id -u leaps &>/dev/null || useradd -s /bin/bash leaps
echo leaps:leaps | chpasswd

cp ./leaps /usr/sbin/leaps
cp ./config/init.d/leaps /etc/init.d/leaps

if [ ! -f /etc/leaps.js ]; then
	cp ./config/leaps_all_fields.js /etc/leaps.js
fi

chmod 755 /etc/init.d/leaps
chown root:root /etc/init.d/leaps

sudo update-rc.d leaps defaults
sudo update-rc.d leaps enable
