package database

import (
	"log"
)

func migrate() error {
	tables := []string{
		`CREATE TABLE IF NOT EXISTS settings (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS conversations (
			id TEXT PRIMARY KEY,
			title TEXT NOT NULL DEFAULT '新会话',
			message_count INTEGER DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS chat_history (
			id TEXT PRIMARY KEY,
			conversation_id TEXT NOT NULL,
			role TEXT NOT NULL CHECK(role IN ('user', 'assistant', 'system', 'tool')),
			content TEXT NOT NULL,
			tool_calls TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (conversation_id) REFERENCES conversations(id) ON DELETE CASCADE
		)`,
		`CREATE TABLE IF NOT EXISTS mcp_connections (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			type TEXT NOT NULL DEFAULT 'sse',
			command TEXT NOT NULL DEFAULT '',
			args TEXT NOT NULL DEFAULT '',
			sse_url TEXT NOT NULL DEFAULT '',
			status TEXT NOT NULL DEFAULT 'disconnected',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS prompts (
			id TEXT PRIMARY KEY,
			title TEXT NOT NULL,
			content TEXT NOT NULL,
			category TEXT NOT NULL DEFAULT 'custom',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
	}

	for _, ddl := range tables {
		if _, err := DB.Exec(ddl); err != nil {
			return err
		}
	}

	if err := seedDefaultSettings(); err != nil {
		return err
	}

	if err := seedDefaultMCPConnections(); err != nil {
		return err
	}

	if err := seedDefaultPrompts(); err != nil {
		return err
	}

	log.Println("[DB] Migration completed")
	return nil
}

func seedDefaultMCPConnections() error {
	var count int
	DB.QueryRow("SELECT COUNT(*) FROM mcp_connections").Scan(&count)
	if count > 0 {
		return nil
	}
	defaults := []struct {
		id, name, mtype, command, args, sseURL string
	}{
		{"blender", "Blender", "sse", "./data/mcp_bin/blender-mcp.exe", "", "http://localhost:56500/sse"},
		{"gimp", "GIMP", "sse", "./data/mcp_bin/gimp-mcp.exe", "", "http://localhost:56600/"},
	}
	for _, d := range defaults {
		_, err := DB.Exec(
			`INSERT OR IGNORE INTO mcp_connections (id, name, type, command, args, sse_url, status)
			 VALUES (?, ?, ?, ?, ?, ?, 'disconnected')`,
			d.id, d.name, d.mtype, d.command, d.args, d.sseURL,
		)
		if err != nil {
			return err
		}
	}
	log.Println("[DB] Default MCP connections seeded:", len(defaults))
	return nil
}

func seedDefaultPrompts() error {
	var count int
	DB.QueryRow("SELECT COUNT(*) FROM prompts").Scan(&count)
	if count > 0 {
		return nil
	}
	defaults := []struct {
		title, content, category string
	}{
		{"Blender 建模专家", "你是 Blender 3D 建模专家。精通多边形建模、曲面建模、雕刻、拓扑优化。\n你可以调用 Blender MCP 工具来执行以下任务：\n- 创建和修改 3D 模型\n- 执行 Blender Python 代码进行高级建模\n- 导入 Polyhaven/Sketchfab 资产\n- 获取场景和物体信息\n\n当用户请求建模相关任务时，主动使用工具完成，不要只给建议。", "blender"},
		{"Blender 渲染师", "你是 Blender 渲染专家，精通 Cycles 和 Eevee 渲染引擎。\n你可以调用 Blender MCP 工具来执行以下任务：\n- 设置材质和纹理\n- 导入 HDR 环境贴图\n- 调整渲染设置\n- 获取视口截图预览\n- 执行 Python 代码自定义渲染\n\n当用户需要渲染、出图、材质设置时，主动使用工具完成。", "blender"},
		{"Blender 场景搭建师", "你是 Blender 场景搭建专家。擅长从零构建完整 3D 场景。\n你可以调用 Blender MCP 工具：\n- 下载并导入 Polyhaven 资产（模型、材质、HDR）\n- 从 Sketchfab 导入模型\n- 执行 Python 代码生成程序化场景\n- 获取场景信息并调整布局\n\n用户需要搭建场景、放置物体、布置环境时，主动使用工具。", "blender"},
		{"GIMP 海报设计师", "你是 GIMP 海报设计专家。精通海报排版、色彩搭配、文字设计。\n你可以调用 GIMP MCP 工具来执行以下任务：\n- 创建新画布并设置尺寸\n- 添加和编辑文字图层\n- 绘制矩形、椭圆等形状\n- 填充颜色和渐变\n- 应用投影、模糊等效果\n- 导出为图片文件\n\n当用户需要设计海报、传单、封面时，主动使用工具完成整个设计流程。", "gimp"},
		{"GIMP 修图师", "你是 GIMP 修图专家，精通照片修饰、色彩校正、瑕疵修复。\n你可以调用 GIMP MCP 工具：\n- 调整亮度对比度、色阶、曲线\n- 调整色相饱和度\n- 应用模糊、锐化、降噪\n- 裁剪和旋转图像\n- 反转颜色、去色\n\n当用户需要修图、调色、美化照片时，主动使用工具完成。", "gimp"},
	}
	for _, d := range defaults {
		_, err := DB.Exec(
			`INSERT OR IGNORE INTO prompts (id, title, content, category) VALUES (hex(randomblob(16)), ?, ?, ?)`,
			d.title, d.content, d.category,
		)
		if err != nil {
			return err
		}
	}
	log.Println("[DB] Default prompts seeded:", len(defaults))
	return nil
}

func seedDefaultSettings() error {
	defaults := map[string]string{
		"deepseek.api_key":  "",
		"deepseek.base_url": "https://api.deepseek.com",
		"deepseek.model":    "deepseek-v4-flash",
		"blender.work_dir":  "",
	}
	for k, v := range defaults {
		_, err := DB.Exec(
			`INSERT OR IGNORE INTO settings (key, value) VALUES (?, ?)`,
			k, v,
		)
		if err != nil {
			return err
		}
	}
	return nil
}
