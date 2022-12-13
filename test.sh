go build -o "test/kade"

cd "test"
startPort=30303
clientCount=10

mkdir -p "./bootnode"
cp ../static/* ./bootnode/
nohup ./kade -p $startPort -k "./bootnode" >> "output.log" &

# Root Node
for ((i=1;i<=$clientCount;i++)) do
  port=$(expr $i + $startPort)
  nohup ./kade -p $port -k "$i">> "output.log" &
  echo $port
done

echo "Start Done"

while true
do
read -n2 -p "Quit [Y/N]?" answer
  case $answer in
  (Y | y)
        pkill -f kade
        exit 0
        ;;
  esac
done