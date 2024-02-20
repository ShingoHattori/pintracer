# pintracer

tracerouteをしてみつけたホストに対して，個別のwindowからpingを飛ばすのはあまりにも非効率なので，tracerouteで見つけたホストの全てに対して，単一のコマンドによって常時pingを飛ばし，その結果を見やすく表示するためのコード．


---

# 使い方

ソケットを触りに行くので，root権限が必要．

タイムアウトしたホストについてはIPアドレスを引けないので，表示しない．ホスト名はFQDNもIPアドレスも対応．

```
$ sudo go run main.go ftp.tsukuba.wide.ad.jp
Tracing route to  ftp.tsukuba.wide.ad.jp
Error: failed to read reply: read ip4 0.0.0.0: i/o timeout at 3 
Error: failed to read reply: read ip4 0.0.0.0: i/o timeout at 4 
Error: failed to read reply: read ip4 0.0.0.0: i/o timeout at 5 
Error: failed to read reply: read ip4 0.0.0.0: i/o timeout at 6 
[{1 133.51.77.253 } {2 130.158.4.2 } {3  } {4  } {5  } {6  } {7 203.178.132.80 }]
1: 133.51.77.253 : Recv echo reply
2: 130.158.4.2 : 
3:  : 
4:  : 
5:  : 
6:  : 
7: 203.178.132.80 : 
--------------------------------------------------
1: 133.51.77.253 : Recv echo reply
2: 130.158.4.2 : 
3:  : 
4:  : 
5:  : 
6:  : 
7: 203.178.132.80 : Recv echo reply
```