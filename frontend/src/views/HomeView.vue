<script setup lang="ts">
import { onMounted, ref } from 'vue';
import type { Ref } from 'vue'

import L from 'leaflet'
import 'leaflet/dist/leaflet.css'
import axios from 'axios'


import '@fortawesome/fontawesome-free/css/fontawesome.css'
import '@fortawesome/fontawesome-free/css/brands.css'
import '@fortawesome/fontawesome-free/css/regular.css'
import '@fortawesome/fontawesome-free/css/solid.css'

interface Pharmacy {
  id: string
  name: string
  postID: string
  address: string
  telephone: string
  fax: string
  point: number
  lat: number
  lon: number
}

interface Marker {
  marker: L.Marker
  pharmacy: Pharmacy
  label: L.Marker
}

const isMenu = ref(false)
const message = ref("")
let map: L.Map
let pharmacies: Pharmacy[]
let markers: Ref<Marker[]> = ref([])

const colorIcon = (color: string): L.Icon => {
  return new L.Icon({
    iconUrl: `images/marker-icon-2x-${color}.png`,
    shadowUrl: 'images/marker-shadow.png',
    iconSize: [25, 41],
    iconAnchor: [12, 41],
    popupAnchor: [1, -34],
    shadowSize: [41, 41]
  })
}

const setupMap = async () => {
  map = L.map('map', {
    zoomControl: false,
  })
  L.tileLayer('https://mt1.google.com/vt/lyrs=r&x={x}&y={y}&z={z}', {
    attribution: "<a href='https://developers.google.com/maps/documentation' target='_blank'>Google Map</a>"
  }).addTo(map)

  L.control.zoom({
    position: 'bottomright'
  }).addTo(map);

  let lat = 35.652832, lon = 139.839478
  if ('geolocation' in navigator) {
    let opt = {
      'enableHighAccuracy': true,
      'timeout': 10000,
      'maximumAge': 0,
    }
    try {
      const pos = await new Promise<GeolocationPosition>((resolve, reject) => {
        navigator.geolocation.getCurrentPosition(resolve, reject, opt)
      });
      lat = pos.coords.latitude
      lon = pos.coords.longitude
    } catch (e) {
      console.log('failed to get position', e)
    }
  }
  map.setView([lat, lon], 17)
  const label3 = L.divIcon({
    iconSize: [125, 23],
    iconAnchor: [-15, 20],
    className: 'text-md font-bold border-none',
    html: `<span class="bg-red-500 text-white text-lg p-1">現在地</span>`,
  });
  L.marker([lat, lon], { icon: colorIcon('red') }).addTo(map)
  L.marker([lat, lon], { icon: label3 }).addTo(map)
}

const displayPharmacies = () => {
  for (const m of markers.value) {
    m.marker.remove()
    m.label.remove()
  }
  markers.value = []
  const bound = map.getBounds()
  let cnt = 0
  for (const p of pharmacies) {
    if (!bound.contains([p.lat, p.lon])) {
      continue
    }
    const sames = markers.value.filter((v) => v.pharmacy.lat === p.lat && v.pharmacy.lon === p.lon)
    const label = L.divIcon({
      iconSize: [125, 23],
      iconAnchor: [-15, 20],
      className: 'text-md font-bold border-none',
      html: `<span class="bg-sky-500 text-white text-xs p-0.5">${p.point * 10}円</span>`,
    });
    const m: Marker = {
      pharmacy: p,
      marker: L.marker([p.lat + sames.length * 0.0001, p.lon], { icon: colorIcon('green') }).addTo(map),
      label: L.marker([p.lat + sames.length * 0.0001, p.lon], { icon: label }).addTo(map),
    }
    markers.value.push(m)
    if (++cnt >= 100) {
      break
    }
  }
  if (cnt >= 100) {
    message.value = '薬局数が１００を超えましたので全ては表示しません。'
  } else {
    message.value = ''
  }
  markers.value.sort((a, b) => a.pharmacy.point - b.pharmacy.point)
  for (let i = 0; i < markers.value.length; i++) {
    const c = markers.value[i]
    const p = c.pharmacy
    const label2 = L.divIcon({
      iconSize: [125, 23],
      iconAnchor: [-15, 20],
      className: 'text-md font-bold border-none',
      html: `<span class="bg-red-500 text-white text-md p-0.5">${p.point * 10}円(最安）</span>`,
    });
    c.marker.setIcon(colorIcon('gold'))
    const loc = c.label.getLatLng()
    c.label.remove()
    c.label = L.marker([loc.lat, loc.lng], { icon: label2 }).addTo(map)
    if (markers.value.length > i + 1 && markers.value[i + 1].pharmacy.point !== p.point) {
      break
    }
  }
}

onMounted(async () => {
  await setupMap()
  const resp = await axios.get<Pharmacy[]>("/pharmacy.json")
  pharmacies = resp.data
  displayPharmacies()

  map.on('moveend', (e) => {
    displayPharmacies()
  })
  map.on('zoomend', (e) => {
    displayPharmacies()
  })
})
</script>

<template>
  <main>
    <div>
      <div :class="{ 'opacity-60': isMenu }">
        <div id="map" class="w-screen h-screen z-0"></div>
      </div>
      <div class="absolute inset-4 pb-1 pl-1.5 pt-0.5 w-10 h-10 z-1  bg-white border border-1 border-black border-solid"
        @click="isMenu = !isMenu">
        <i class="fa-solid fa-bars text-3xl"></i>
      </div>
      <div v-show="message"
        class="absolute top-0 right-0 p-1.5 w-400 h-8 z-1 font-bold text-white bg-orange-400 overflow-hidden shadow-2xl m-3">
        {{ message }}
      </div>
      <transition enter-active-class="transition ease-in-out duration-300 transform"
        enter-from-class="-translate-x-full md:-translate-x-96" enter-to-class="translate-x-0"
        leave-from-class="translate-x-0" leave-active-class="transition ease-in-out duration-300 transform"
        leave-to-class="-translate-x-full" appear>
        <div v-show="isMenu"
          class="absolute inset-0 p-1.5 w-screen md:w-96 h-full z-1  bg-white overflow-hidden shadow-2xl">
          <div class="grid grid-cols-12 mt-1">
            <div class="col-start-2 col-span-7 text-2xl font-bold text-gray-500">格安薬局マップ</div>
            <i class="col-end-12 col-span-1 fa-solid fa-xmark text-3xl ml-5 text-gray-400"
              @click="isMenu = !isMenu"></i>
          </div>
          <div class="mt-5 p-1">
            <div class="text-center bg-sky-500 text-white font-bold">
              <i class="fa-solid fa-award mr-3"></i>格安ランキング10位
            </div>
            <ul class="mt-2 pb-4 border-b-2 border-b-gray-300">
              <li v-for="(m, i) in markers" :key="m.pharmacy.id">
                <div v-if="i < 10" class="grid grid-cols-12 text-sm mb-0.5" :class="{ 'bg-pink-100': i % 2 == 0 }">
                  <span class="col-span-1 bg-pink-500 text-white text-center font-bold"> {{ i + 1 }}</span>
                  <span class="col-span-9 ml-2 font-bold"> {{ m.pharmacy.name }}</span>
                  <span class="col-span-2 text-right bg-gray-400 text-white font-bold pr-1">
                    {{ m.pharmacy.point * 10 }} 円
                  </span>
                </div>
              </li>
            </ul>
            <div class="mt-2 pb-4 pt-4 border-b-2 border-b-gray-300">
              <div class="text-center bg-sky-500 text-white font-bold">
                <i class="fa-regular fa-circle-question mr-2"></i>FAQ
              </div>
              <ul>
                <li class="bg-pink-100">これは何？</li>
                <li>これは何？</li>
              </ul>
            </div>
          </div>
        </div>
      </transition>
    </div>
  </main>
</template>

