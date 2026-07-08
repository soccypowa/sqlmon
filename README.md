# sqlmon

## Bakgrund

Vad använder jag egentligen SSMS till? Vad skulle jag sakna om jag bara körde från kommandoraden?

Troligen skulle det vara *activity monitor* att se sessionerna, vilka frågor de kör och eventuella dyra frågor.

Sagt och gjort, sqlmon, ett **TUI** som gör just det som *activity monitor* gör!

## Usage

sqlmon --conn sqlserver://user:password@host[:port][/instance]

sqlmon --conn "sqlserver://myserver?trusted_connection=true

sqlmon --conn "sqlserver://myserver:1433" --interval 10s

För ytterligare information ta en titt på: https://github.com/microsoft/go-mssqldb