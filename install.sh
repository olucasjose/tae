#!/bin/sh
set -e

echo "Compilando tae..."
go build -o tae main.go

# $PREFIX é uma variável de ambiente nativa e exclusiva do Termux
if [ -n "$PREFIX" ] && [ -d "$PREFIX/bin" ]; then
    echo "Ambiente Termux detectado."
    DEST="$PREFIX/bin/tae"
    mv tae "$DEST"
    chmod +x "$DEST"
else
    echo "Ambiente Linux padrão detectado."
    DEST="/usr/local/bin/tae"
    echo "Isso requer privilégios de root para gravar em $DEST"
    sudo mv tae "$DEST"
    sudo chmod +x "$DEST"
fi

echo "Sucesso! 'tae' instalado em $DEST."
echo "Você já pode executar o comando 'tae' de qualquer lugar."
