echo "开始编译linux"
GOOS=linux GOARCH=amd64 go build -o bin/shushu-linux
echo "开始编译mac"
GOOS=darwin GOARCH=amd64 go build -o bin/shushu-mac
echo "开始编译windows"
GOOS=windows GOARCH=amd64 go build -o bin/shushu-windows.exe
