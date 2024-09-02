if [ "$USER" != "root" ]
then
    echo "Please run this as root or with sudo"
    exit 2
fi
systemctl stop InverterValues

cp dist/amd64/InverterValues /usr/bin
chmod +x /usr/bin/InverterValues

systemctl start InverterValues

echo "Done"
