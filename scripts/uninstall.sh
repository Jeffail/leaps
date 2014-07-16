#!/bin/bash

if [ -f /etc/init.d/leaps ]; then
	/etc/init.d/leaps stop
fi

userdel leaps

rm /usr/sbin/leaps
rm /etc/init.d/leaps

update-rc.d -f leaps remove
