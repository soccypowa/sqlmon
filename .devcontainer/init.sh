#wait for the SQL Server to come up
sleep 30s

echo "updating shit"
apt-get update
apt-get upgrade -y
apt install software-properties-common -y

echo "setup tools"
curl https://packages.microsoft.com/keys/microsoft.asc | tee /etc/apt/trusted.gpg.d/microsoft.asc
add-apt-repository "$(wget -qO- https://packages.microsoft.com/config/ubuntu/20.04/prod.list)"
apt-get update
apt-get install sqlcmd

echo "running set up script"
#run the setup script to create the DB and the schema in the DB
/usr/bin/sqlcmd -S $MSSQL_HOSTNAME -U $MSSQL_USER -P $MSSQL_SA_PASSWORD -d master -i dbcreation.sql