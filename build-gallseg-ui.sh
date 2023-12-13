cd ./ui
DOCKER_BUILDKIT=0 docker build -t web-builder -f gallseg.Dockerfile .
container_id=$(docker create web-builder)
docker cp $container_id:/webui/build ./web
curl https://bootstrap.urbit.org/globberv3.tgz | tar xzk
./zod/.run -d
dojo () {
  echo $1
  curl -s --data '{"source":{"dojo":"'"$1"'"},"sink":{"stdout":null}}' http://localhost:12321    
}
hood () {
  curl -s --data '{"source":{"dojo":"+hood/'"$1"'"},"sink":{"app":"hood"}}' http://localhost:12321    
}
mv web zod/work/gallseg
hood "commit %work"
dojo "-garden!make-glob %work /gallseg"
hash=$(ls -1 -c zod/.urb/put | head -1 | sed "s/glob-\\([a-z0-9\\.]*\\).glob/\\1/")
echo "hash=${hash}"
hood "exit"
