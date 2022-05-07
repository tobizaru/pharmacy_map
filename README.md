# 薬局マップ

[薬局マップ](https://tobizaru.github.io/pharmacy_map/)


調剤報酬の調剤基本料付きの薬局マップです。

次の２つの構成から成ります。

* [retriever](/retriever)
   
   厚生局から薬局情報を取得するプログラム（Go言語）

* [frontend](frontend)

  ブラウザ上に薬局マップを表示するフロントエンド（Vue3)

## ライセンス

MITライセンス

## 参考

* [【豆知識】同じ処方箋でも持っていく調剤薬局によって料金に差が出る理由【薬剤師監修】](https://pharmacyassistant.xyz/entry/setsuyaku-yakkyoku-okusuridai/)
* [【薬局マップ】基本料が安い処方箋薬局を地図で探す｜店によって調剤基本料に違いがある？【薬剤師監修】](https://pharmacyassistant.xyz/entry/yasui-yakkyoku-ranking/)


## 謝辞
本ツールでは下記情報及びAPIを使用させていただいております。

* [CSISシンプルジオコーディング実験]( https://geocode.csis.u-tokyo.ac.jp/ )
* 厚生労働省の各厚生局HP 薬局の施設基準の届出状況(URLはretriever/xls_url.yml参照)
