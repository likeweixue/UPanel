# UPanel Web

UPanel 管理后台前端，Vue 3 + Vite，无 UI 框架依赖。

## 设计

- Flarum 2.0 admin SPA 风格：flat / minimalist，固定 header + drawer + Widget 卡片流。
- 渐进披露：抽屉只展示一级入口，36 个功能入口收进可搜索下拉（`/` 或 `Ctrl/Cmd+K`）。
- 自研 BEM-ish 组件：`src/components/ui/` 下的 Button、DropdownSelect、DashboardWidget（UWidget）、
  StatusDot、Modal、Tabs、Toast、Sparkline 等。
- 样式 token：`src/styles/tokens.css`；基础样式：`base.css`；组件：`components.css`；布局：`layout.css`。
- a11y：skip link、`aria-*`、键盘导航、焦点陷阱、`aria-live`、`prefers-reduced-motion`。

## 目录

```
src/
├── api/              axios 封装与接口
├── components/
│   ├── AdminNavDropdown.vue
│   └── ui/           自研组件库
├── composables/      auth / theme / toast / format
├── layouts/          AdminLayout.vue
├── nav/catalog.js    36 个导航入口
├── router/
├── styles/
└── views/            页面
```

## 开发

```bash
npm install
npm run dev        # http://localhost:5173，/api 代理到 :8080
npm run build      # 产出 dist/
```

安全入口由服务端向 `index.html` 注入 `<base href="/entry/">`；前端通过 `<base>` 推导 API 与
router base，因此构建产物需要由 UPanel 主程序托管，而不是用 `file://` 直接打开。
