#!/bin/bash

info() {
		reg=$1
		pids=$(ps x | egrep "$reg" | grep -v grep | awk '{ print $1 }')

		echo "pid|fd|cmd"
		for pid in $pids;do
				name=$(ps -p $pid u | egrep '[0-9]' | awk -F '[0-9]{1,2}:[0-9]{2}[ \t]+[0-9]{1,2}:[0-9]{2}' '{ print $2 }' | egrep -o '[^ ]+.*[^ ]*')
				fdcount=$(( $(ls -l /proc/${pid}/fd | wc -l) - 1 ))
				echo "$pid|$fdcount|$name"
		done
}

case $1 in
		info)
				info 'exposer|shadowsocks' | column -t -s '|'
				;;
		watch)
				watch -n 1 $0 info
				;;
		request)
				i=0
				while [ $? == 0 ];do
						curl -s --socks5 localhost:1080 http://localhost:6060 1>/dev/null
						i=$(($i+1))
						echo $i
				done
				;;
		version)
				echo 0.1
				;;
		help)
				echo "$0 [info|watch|version|help]"
				;;
		*)
				if [ $# == 0 ];then
						$0 watch
				else
						$0 help
				fi
				;;
esac

