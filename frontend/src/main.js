// main.js — Svelte application entry point.
// Mounts the root App component and wires Wails runtime events.

import { mount } from "svelte";
import App from "./App.svelte";
import "./themes/default-dark.css";

const app = mount(App, { target: document.getElementById("app") });

export default app;
