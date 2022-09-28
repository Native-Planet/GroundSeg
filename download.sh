# install curl
sudo apt-get install -y curl

DIR='/tmp/nativeplanet'
IFS='"'
DOWNLOAD_URL=$(curl -s https://api.github.com/repos/nallux-dozryl/GroundSeg/releases/latest | grep tarball_url)
RELEASE_ID=$(curl -s https://api.github.com/repos/nallux-dozryl/GroundSeg/releases/latest | grep \"id\")
VERSION=$(curl -s https://api.github.com/repos/nallux-dozryl/GroundSeg/releases/latest | grep tag_name)

FILE_NAME="groundseg.tar.gz"
UNTAR_DIR="groundseg"

# Create temporary dir
sudo rm -r $DIR
mkdir -p $DIR

# Download latesst release
read -ra arr <<< "$DOWNLOAD_URL"
wget -O $DIR/$FILE_NAME ${arr[3]} 

cd $DIR

# untar
tar xzvf $FILE_NAME -C $DIR
rm $FILE_NAME
mkdir -p $UNTAR_DIR
mv na*/* $UNTAR_DIR 
rm -r na*
cd $UNTAR_DIR

# install
IFS=' '
read -ra arrtwo <<< "$RELEASE_ID"

IFS=','
read -ra arrthree <<< "${arrtwo[1]}"

IFS='"'
read -ra arrfour <<< "${VERSION}"

./build.sh

sudo echo ${arrfour[3]} > build/version
sudo echo ${arrthree[0]} > build/release_id

./install.sh
