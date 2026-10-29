# UPanel 主题目录

把主题文件夹放到这里即可被面板识别，无需重新编译：

```text
themes/
└── flarum/
    ├── theme.json   # 可选，主题名、描述、dark、themeColor、radius、swatch
    └── theme.css    # 必需，至少包含 html[data-theme="<文件夹名>"] 的 token
```

主题文件夹名就是主题 ID，只能使用小写字母、数字和短横线。

`theme.json` 示例：

```json
{
  "name": "Flarum",
  "description": "Flarum 风格：系统字体、4px 圆角、灰蓝配色",
  "version": "1.0.0",
  "author": "UPanel",
  "dark": false,
  "themeColor": "#f7f7f7",
  "radius": 4,
  "swatch": ["#ffffff", "#536f90"]
}
```

`theme.css` 通过 `html[data-theme="flarum"]` 覆盖 UPanel 的 CSS 变量即可。
新增或修改主题后刷新页面，在「设置 → 外观」中即可看到。
