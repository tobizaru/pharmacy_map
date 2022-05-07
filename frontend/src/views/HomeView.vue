<script setup lang="ts">
//マップ表示（メイン）

import { onMounted, ref, watch } from "vue";
import type { Ref } from "vue";

import L from "leaflet";
import "leaflet/dist/leaflet.css";
import axios from "axios";

import "@fortawesome/fontawesome-free/css/fontawesome.css";
import "@fortawesome/fontawesome-free/css/brands.css";
import "@fortawesome/fontawesome-free/css/regular.css";
import "@fortawesome/fontawesome-free/css/solid.css";

//モーダルダイアログコンポーネント
import ModalDialog from "../components/modalDialog.vue";

//薬局情報（JSONから取得）
interface Pharmacy {
  id: string;
  name: string; //薬局名
  post_id: string; //郵便番号
  address: string; //住所
  telephone: string; //電話番号
  fax: string; //FAX番号
  point: number; //調剤基本料
  lat: number; //緯度
  lon: number; //経度
}

//マップ上のマーカ情報
interface Marker {
  pharmacy: Pharmacy; //薬局情報
  marker: L.Marker; //マーカー
  label: L.Marker; //基本料マーカー
}

const isMenu = ref(false); //左メニュースイッチ
const message = ref(""); //下エラーメッセージ
//モーダルダイアログ各スイッチ
const dialog: { [key: string]: Ref<boolean> } = {
  isDisclaimer: ref(false), //免責ダイアログ
  isExplain: ref(false), //格安薬局マップについて
  isNotice: ref(false), //ご注意
};
const markers: Ref<Marker[]> = ref([]); //全マーカー
const selected: Ref<Pharmacy | undefined> = ref(); //選択された薬局

const rate = ref(0.3); //自己負担割合
const search = ref(""); //検索ワード
const searchFilter: Ref<Pharmacy[]> = ref([]); //検索ワードを含む薬局、サジェスト用

let pharmacies: Pharmacy[] = []; //全薬局
let map: L.Map; //leafletマップ
let layer = L.layerGroup([]); //マーカー用グループ

//カラー指定のマーカーアイコン
const colorIcon = (color: string): L.Icon => {
  return new L.Icon({
    iconUrl: `images/marker-icon-2x-${color}.png`,
    shadowUrl: "images/marker-shadow.png",
    iconSize: [25, 41],
    iconAnchor: [12, 41],
    popupAnchor: [1, -34],
    shadowSize: [41, 41],
  });
};

//マップ初期化
const setupMap = async () => {
  //マップ作成
  map = L.map("map", {
    zoomControl: false,
  });
  //ライセンス情報
  L.tileLayer("https://mt1.google.com/vt/lyrs=r&x={x}&y={y}&z={z}", {
    attribution:
      "<a href='https://developers.google.com/maps/documentation' target='_blank'>Google Map</a>",
  }).addTo(map);
  layer.addTo(map);

  //ズーム制御を右下に
  L.control
    .zoom({
      position: "bottomright",
    })
    .addTo(map);

  //デフォルトの現在地（東京）
  let lat = 35.652832;
  let lon = 139.839478;

  //Geolocation APIで現在位置取得
  if ("geolocation" in navigator) {
    let opt = {
      enableHighAccuracy: true,
      timeout: 10000,
      maximumAge: 0,
    };
    try {
      // eslint-disable-next-line no-undef
      const pos = await new Promise<GeolocationPosition>((resolve, reject) => {
        navigator.geolocation.getCurrentPosition(resolve, reject, opt);
      });
      lat = pos.coords.latitude;
      lon = pos.coords.longitude;
    } catch (e) {
      console.log("failed to get position", e);
    }
  }
  //現在位置を中心にsしてラベル付け
  map.setView([lat, lon], 17);
  const label3 = L.divIcon({
    iconSize: [125, 23],
    iconAnchor: [-15, 20],
    className: "text-md font-bold border-none",
    html: `<span class="bg-red-500 text-white text-lg p-1">現在地</span>`,
  });
  L.marker([lat, lon], { icon: colorIcon("red") }).addTo(map);
  L.marker([lat, lon], { icon: label3 }).addTo(map);
};

//マーカ作成
const newMarker = (p: Pharmacy) => {
  //同位置にある薬局取得
  const sames = markers.value.filter(
    (v) => v.pharmacy.lat === p.lat && v.pharmacy.lon === p.lon
  );

  //緑色でマーカー作成、他の薬局が同位置にあれば重さらないように微妙に位置を変える
  const marker = L.marker([p.lat + sames.length * 0.0001, p.lon], {
    icon: colorIcon("green"),
  }).addTo(layer);
  marker.on("click", () => {
    markerSelected(p);
  });

  //基本料表示アイコン追加
  const priceLabel = L.divIcon({
    iconSize: [125, 23],
    iconAnchor: [-15, 20],
    className: "text-md font-bold border-none",
    html:
      "<span class='bg-sky-500 text-white text-xs p-0.5 shadow-2xl shadow-gray-200'>" +
      `${p.point * 10 * rate.value}円</span>`,
  });
  const label = L.marker([p.lat + sames.length * 0.0001, p.lon], {
    icon: priceLabel,
  }).addTo(layer);

  //選択中の薬局なら薬局名をポップアップ
  if (p.id === selected.value?.id) {
    marker.bindPopup(p.name).openPopup();
    setTimeout(() => {
      marker.on("popupclose", () => {
        selected.value = undefined;
      });
    }, 1000);
  }

  //マーカー情報を返す
  const m: Marker = {
    pharmacy: p,
    marker: marker,
    label: label,
  };
  return m;
};

//表示範囲内の薬局の表示
const displayPharmacies = () => {
  //現表示のイベントと表示を全削除
  layer.eachLayer((m) => {
    m.off();
  });
  layer.clearLayers();
  markers.value = [];

  //表示範囲内の薬局が100以上ならメッセージ表示してなにもしない（重くなるので）
  const bound = map.getBounds();
  let cnt = 0;
  for (const p of pharmacies) {
    if (bound.contains([p.lat, p.lon])) {
      cnt++;
    }
  }
  if (cnt >= 100) {
    message.value = "表示範囲を絞ってください。";
    return;
  }

  //範囲内の全薬局のマーカー作成
  message.value = "";
  for (const p of pharmacies) {
    if (!bound.contains([p.lat, p.lon])) {
      continue;
    }
    const m = newMarker(p);
    markers.value.push(m);
  }
  //基本料順に並べ替え
  markers.value.sort((a, b) => a.pharmacy.point - b.pharmacy.point);

  //最安薬局（複数あり）に最安用マーカを表示
  for (let i = 0; i < markers.value.length; i++) {
    const c = markers.value[i];
    const p = c.pharmacy;
    const label2 = L.divIcon({
      iconSize: [125, 23],
      iconAnchor: [-15, 20],
      className: "text-md font-bold border-none",
      html:
        "<span class='bg-red-500 text-white text-md p-0.5'>" +
        `${p.point * 10 * rate.value}円(最安）</span>`,
    });
    c.marker.setIcon(colorIcon("gold"));
    const loc = c.label.getLatLng();
    //現在の基本料マーカを一旦削除し再設定
    layer.removeLayer(c.label);
    c.label = L.marker([loc.lat, loc.lng], { icon: label2 }).addTo(layer);
    //次の薬局が最安でなければ終了
    if (
      markers.value.length > i + 1 &&
      markers.value[i + 1].pharmacy.point !== p.point
    ) {
      break;
    }
  }
};

onMounted(async () => {
  //マップ初期化
  await setupMap();
  //全薬局情報取得
  const resp = await axios.get<Pharmacy[]>("pharmacy.json");
  pharmacies = resp.data;
  //範囲内薬局表示
  displayPharmacies();

  //マップを動かしたときに薬局再表示
  map.on("moveend", () => {
    displayPharmacies();
  });
});

//全モーダルダイアログ表示OFF
const offModal = () => {
  for (const k of Object.keys(dialog)) {
    dialog[k].value = false;
  }
};

//全モーダルダイアログがオフか
const isAllModalOff = () => {
  let r = true;
  for (const k of Object.keys(dialog)) {
    r &&= !dialog[k].value;
  }
  return r;
};

//薬局が選択されたら、その薬局を中心に
const markerSelected = (m: Pharmacy) => {
  selected.value = m;
  map.flyTo([m.lat, m.lon], 17);
};

//自己負担割合が設定されたら薬局マーカ再表示
watch(rate, () => {
  displayPharmacies();
});

//検索が入力されている途中ならサジェスト（searchFilter)を設定
const handleInput = () => {
  if (!search.value) {
    searchFilter.value = [];
    return;
  }
  //入力されたワードを含む薬局名を取得
  searchFilter.value = pharmacies.filter((v) => v.name.includes(search.value));
};

//検索されたら薬局を探し選択扱い（中心に表示）
const handleSearch = () => {
  if (!search.value) {
    return;
  }
  const p = pharmacies.filter((v) => v.name === search.value);
  if (!p || !p.length) {
    return;
  }
  selected.value = p[0];
  //遠い場合があるのでflyTo(アニメーション)はしない
  map.panTo([p[0].lat, p[0].lon]);
};
</script>

<template>
  <main>
    <div>
      <!-- マップ表示、モーダル表示中は透明度50％に -->
      <div :class="{ 'opacity-50': !isAllModalOff() }">
        <div id="map" class="w-screen h-screen z-0"></div>
      </div>

      <!-- メニューボタン、薬局検索バー、自己負担割合選択バー表示 -->
      <div class="absolute inset-2 pr-3 h-10 w-full md:w-96 z-1">
        <div
          class="border border-1 border-gray-300 border-solid flex flex-row gap-2 shadow-xl rounded-lg bg-white"
        >
          <div @click="isMenu = !isMenu">
            <i class="fa-solid fa-bars text-xl pl-2 pt-1"></i>
          </div>
          <input
            v-model="search"
            list="pharmacy_list"
            class="border-l-2 pl-1 w-64"
            placeholder="薬局を検索"
            @input="handleInput"
            @change="handleSearch"
          />
          <datalist v-if="!isMenu" id="pharmacy_list">
            <option
              v-for="p in searchFilter"
              :key="p.id"
              :value="p.name"
            ></option>
          </datalist>
          <i class="fa-solid fa-magnifying-glass text-xl mt-1 ml-7"></i>
        </div>
        <div class="mt-1">
          <select
            v-model="rate"
            class="bg-white rounded-lg shadow-xl border-2 border-gray-300"
            name="rate"
          >
            <option disabled value="">自己負担割合</option>
            <option value="0.3">3割</option>
            <option value="0.1">1割</option>
            <option value="0.2">2割</option>
            <option value="1">全額</option>
          </select>
        </div>
      </div>

      <!-- メッセージ表示 -->
      <div
        v-show="message"
        class="absolute bottom-0 left-0 p-1.5 w-full md:w-96 h-8 z-1 font-bold text-white bg-orange-400 overflow-hidden shadow-xl text-sm z-10"
      >
        {{ message }}
      </div>

      <!-- 下に薬局情報表示、下から上にtransition -->
      <transition
        enter-active-class="transition ease-in-out duration-300 transform"
        enter-from-class="translate-y-full"
        enter-to-class="translate-y-0"
        leave-active-class="transition ease-in-out duration-300 transform"
        leave-from-class="translate-y-0"
        leave-to-class="translate-y-full"
      >
        <div
          v-show="selected"
          class="absolute inset-x-0 bottom-0 bg-white pt-5 pr-5 pl-5 w-screen md:w-96 mx-auto shadow-2xl h-48"
        >
          <div class="text-center mb-2 bg-sky-500 text-white font-bold">
            <i class="fa-solid fa-circle-info mr-3"></i>
            薬局情報
          </div>
          <div>
            <div class="font-bold mb-1">
              {{ selected?.name }}
            </div>
            <div class="text-sm">
              <div>
                <i class="fa-solid fa-location-dot pr-2"></i>
                〒 {{ selected?.post_id }}
              </div>
              <div class="pl-6">
                {{ selected?.address }}
              </div>
            </div>
            <div class="pt-1 text-sm">
              <i class="fa-solid fa-phone pr-2"></i>
              {{ selected?.telephone }}
            </div>
            <div class="pt-1 text-sm">
              <i class="fa-solid fa-yen-sign pr-3"></i>
              {{ selected?.point * 10 * rate }} 円
            </div>
          </div>
        </div>
      </transition>

      <!-- 左にランキングとリンク表示、左から右にtransition -->
      <transition
        enter-active-class="transition ease-in-out duration-300 transform"
        enter-from-class="-translate-x-full md:-translate-x-96"
        enter-to-class="translate-x-0"
        leave-from-class="translate-x-0"
        leave-active-class="transition ease-in-out duration-300 transform"
        leave-to-class="-translate-x-full"
      >
        <div
          v-show="isMenu"
          class="absolute inset-0 p-1.5 w-screen md:w-96 h-screen z-1 bg-white overflow-hidden shadow-2xl"
        >
          <!-- タイトル -->
          <div class="grid grid-cols-12 mt-1">
            <div class="col-start-2 col-span-7 text-xl font-bold">
              <i class="fa-solid fa-capsules mr-3"></i>格安薬局マップ
            </div>
            <i
              class="col-end-12 col-span-1 fa-solid fa-xmark text-3xl ml-5 text-gray-400"
              @click="isMenu = !isMenu"
            ></i>
          </div>

          <!-- ランキング表示 -->
          <div class="mt-2">
            <div class="mt-2 p-1 pb-4 border-b-2 border-b-gray-300">
              <div v-show="markers.length">
                <div class="text-center mb-2 bg-sky-500 text-white font-bold">
                  <i class="fa-solid fa-award mr-3"></i>
                  調剤基本料格安ランキング
                </div>
                <ul class="">
                  <li v-for="(m, i) in markers" :key="m.pharmacy.id">
                    <div
                      v-if="i < 10"
                      class="grid grid-cols-12 text-sm mb-1"
                      :class="{ 'bg-gray-100': i % 2 == 0 }"
                    >
                      <span
                        class="col-span-1 bg-sky-500 text-white text-center font-bold"
                      >
                        {{ i + 1 }}</span
                      >
                      <span
                        class="col-span-9 ml-2"
                        @click="
                          isMenu = false;
                          markerSelected(m.pharmacy);
                        "
                      >
                        {{ m.pharmacy.name }}
                      </span>
                      <span
                        class="col-span-2 text-right text-black font-bold pr-1"
                      >
                        {{ m.pharmacy.point * 10 * rate }} 円
                      </span>
                    </div>
                  </li>
                </ul>
              </div>
            </div>

            <!-- マップ説明ダイアログへのリンク -->
            <div class="mt-2">
              <div class="text-gray-700">
                <i class="fa-solid fa-circle-question mr-3 text-lg"></i>
                <span
                  class="text-sm"
                  @click="
                    offModal();
                    dialog.isExplain.value = true;
                  "
                  >格安薬局マップについて</span
                >
              </div>
            </div>

            <!-- ご注意ダイアログへのリンク -->
            <div class="mt-2">
              <div class="text-gray-700">
                <i class="fa-solid fa-triangle-exclamation mr-3 text-lg"></i>
                <span
                  class="text-sm"
                  @click="
                    offModal();
                    dialog.isNotice.value = true;
                  "
                  >ご注意</span
                >
              </div>
            </div>

            <!-- ホームへのリンク -->
            <div class="mt-2">
              <a href="/" class="text-gray-700">
                <i class="fa-solid fa-house mr-3 text-lg"></i>
                <span class="text-sm">格安薬局マップ ホーム</span>
              </a>
            </div>

            <!-- 免責ダイアログへのリンク -->
            <div class="mt-2">
              <div class="text-gray-700">
                <i class="fa-solid fa-circle-exclamation mr-3 text-lg"></i>
                <span
                  class="text-sm"
                  @click="
                    offModal();
                    dialog.isDisclaimer.value = true;
                  "
                  >出典・免責</span
                >
              </div>
            </div>

            <!-- githubへのリンク -->
            <div class="mt-2 pb-2 border-b-2 border-b-gray-300">
              <a
                href="https://github.com/tobizaru/pharmacy_map"
                class="text-gray-700"
              >
                <i class="fa-brands fa-github mr-3 text-lg"></i>
                <span class="text-sm">ソースファイル（github)</span>
              </a>
            </div>
          </div>
        </div>
      </transition>
    </div>

    <!-- 免責 -->
    <ModalDialog v-model:is-show="dialog.isDisclaimer.value">
      <template #title>
        <i class="fa-solid fa-circle-exclamation mr-3 text-2lg"></i> 免責その他
      </template>
      <template #content>
        <div class="">
          <div>本マップは</div>
          <div class="ml-3">厚生労働省の各厚生局HP</div>
          <div class="ml-3">「施設基準の届出等受理状況一覧」</div>
          <div>を元に作成しています。</div>
          <div class="mt-5 mr-2">
            当該コンテンツに起因してご利用者様および第三者に損害が発生したとしても、当方は一切責任を負いません。
          </div>
        </div>
      </template>
    </ModalDialog>

    <!-- 格安薬局マップについて -->
    <ModalDialog v-model:is-show="dialog.isExplain.value">
      <template #title>
        <i class="fa-solid fa-circle-question mr-3 text-3lg"></i>
        格安薬局マップについて
      </template>
      <template #content>
        <div class="">
          各薬局ごとで異なる「調剤基本料」を元に、表示している地域内の格安薬局がひと目でわかるマップです。
        </div>
        <div>
          詳しくは<a href="/" class="text-blue-500 underline"> ホーム</a>
          を参照してください。
        </div>
      </template>
    </ModalDialog>

    <!-- ご注意ダイアログ -->
    <ModalDialog v-model:is-show="dialog.isNotice.value">
      <template #title>
        <i class="fa-solid fa-triangle-exclamation mr-3 text-2lg"></i>ご注意
      </template>
      <template #content>
        <div class="">
          令和4年4月1日付けで調剤報酬点数が変更となりましたが厚生局からは最新のデータが公開されていません。
          そのため実際の調剤基本料と異なる場合があります。
        </div>
        <div>
          また東北厚生局では薬局情報が全く公開されていませんので、東北地方の薬局情報は閲覧できません。
        </div>
      </template>
    </ModalDialog>
  </main>
</template>
