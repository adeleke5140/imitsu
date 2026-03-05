// @ts-check
import { defineConfig } from 'astro/config';

// https://astro.build/config
export default defineConfig({
  redirects: {
    '/install': 'https://raw.githubusercontent.com/adeleke5140/imitsu/main/install.sh',
  },
});
