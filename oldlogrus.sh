#!/usr/bin/env bash

command -v ag >/dev/null 2>&1 || { echo "*** oldlogrus requires ag (the silver searcher) ***" >&2; echo "install it with homebrew (brew install the_silver_searcher)" >&2; exit 1; }
#git checkout oldlogrus-master
#git merge master

FILES=$(ag -l --nocolor --ignore=./vendor --ignore=./.git --ignore=./oldlogrus.sh sirupsen)

for i in ${FILES}; do
	F="$(pwd)/${i}"
	echo "replacing sirupsen in ${i}"
	sed -i .bak 's/sirupsen/Sirupsen/' ${F}
	rm -f *.bak
done

#git push

#git checkout master