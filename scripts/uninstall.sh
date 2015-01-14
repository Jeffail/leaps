#!/bin/bash

if [ -f /etc/init.d/leaps ]; then
	/etc/init.d/leaps stop

	rm /etc/init.d/leaps
	update-rc.d -f leaps remove
fi

userdel leaps
rm -f /usr/sbin/leaps

echo "Do you wish to keep the /etc/leaps directory? [y]/n"
read answer

if [[ "$answer" != "n" && "$answer" != "N" ]]; then
	rm -rf /etc/leaps
fi
