#!/bin/sh

echo2()
{
	echo "$@" 1>&2
}

usage()
{
	echo2 "usage: apply.sh [-v] database [migration ...]"
	exit 1
}

run()
{
	if test $VERBOSITY -gt 1; then echo "$@"; fi
	"$@"
}

test $# -lt 1 && usage

VERBOSITY=0
VERBOSITYFLAGS="-q"
PGOPTIONS='--client-min-messages=warning'
while test "$1" = "-v"; do
	VERBOSITY=$((VERBOSITY+1))
	VERBOSITYFLAGS="-b"
	PGOPTIONS=""
	shift
done
export PGOPTIONS

DB=$1
shift

case $# in
0)
	FILES=`ls *.sql | grep -v _test`
	;;
*)
	FILES="$@"
	;;
esac

for file in $FILES; do
	printf "Applying '%s'... " "$file"
	if test -f $file; then
		run psql -U postgres $VERBOSITYFLAGS -d "$DB" -c '\set ON_ERROR_STOP' -f "$file" && echo "OK." || echo "FAIL."
	else
		echo2 "FAIL: no such file or directory."
	fi
done
