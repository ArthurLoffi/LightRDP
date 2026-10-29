# Utilização
```
# Nenhum computador a ser acessado (host):
LightRDP # Gera código de acesso
```
```
# Nenhum computador que vai acessar (cliente):
LightRDP --acess CODIGO_DO_HOST # Pede acesso para o host, caso ele aceite, abre uma janela com o desktop dele
```

## Código
O código seria um ID público curto + senha efemera (de uso único), o código sempre será o mesmo, estará relacionado à maquina do usuário. A senha e gerada a cada código gerado pelo host

# Papel do servidor
- Registrar hosts, repassar pedido e SDP/ICE

## Pilha
- Golang
- WebRTC
- WebSocket