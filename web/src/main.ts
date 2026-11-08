import { mount } from 'svelte';
import './app.css';
import App from './App.svelte';
import { initRouter } from './lib/router.svelte';
import { applyTheme } from './lib/theme.svelte';

applyTheme();
initRouter();

const target = document.getElementById('app');
if (!target) throw new Error('#app not found');

export default mount(App, { target });
