# 薬局情報抽出

厚生労働省の各厚生局HPから薬局の施設基準の届出状況を取得し、各薬局の位置情報と調剤報酬の基本料を計算しJSONとして出力します。

## 使用言語

Go言語 バージョン1.18.1 

## 使い方

1. $ git clone https://github.com/tobizaru/pharmacy_map
1. $ cd retriever
1. 調剤報酬をreward.ymlに記載します。
1. 薬局の施設基準の届出状況（EXCELファイルまたはZIPファイル）をxls_urls.ymlに記載します。
1. 本プログラムをそのまま実行します。引数はありません。

    $ go run .

1. pharmacy.jsonが出力されます。

## ライセンス

MITライセンス

## 謝辞
本ツールでは下記情報及びAPIを使用させていただいております。

* [CSISシンプルジオコーディング実験]( https://geocode.csis.u-tokyo.ac.jp/ )
* 厚生労働省の各厚生局HP 薬局の施設基準の届出状況(URLはxls_url.yml参照)
