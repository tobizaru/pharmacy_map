<script setup lang="ts">
//モーダルダイアログの表示

defineProps({
  isShow: Boolean, //表示スイッチ
});
defineEmits<{
  (e: "update:is-show", value: boolean): void; //表示スイッチオフ通知
}>();
</script>

<template>
  <div
    v-show="isShow"
    class="shadow-2xl absolute inset-0 w-full md:w-96 h-96 mx-auto my-auto bg-white"
  >
    <div>
      <div
        class="text-2xl p-3 grid grid-cols-12 border border-b-1 border-b-gray-300 bg-gray-300 rounded"
      >
        <!-- タイトル -->
        <div class="col-start-1 col-span-11">
          <slot name="title"></slot>
        </div>
        <!-- 閉じるボタン -->
        <span
          class="col-end-13 col-span-1 text-right"
          @click="$emit('update:is-show', false)"
        >
          <i class="fa-solid fa-xmark"></i>
        </span>
      </div>
      <!-- コンテント -->
      <div class="text-md p-3 pt-10">
        <slot name="content"></slot>
      </div>
    </div>
  </div>
</template>
