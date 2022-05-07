import { createApp } from "vue";
import App from "./App.vue";
import router from "./router";
import "./assets/index.css"; //tailwind 設定用

const app = createApp(App);

app.use(router);

app.mount("#app");
