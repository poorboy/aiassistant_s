# AI Assistant — User Manual

[**中文**](MANUAL.zh.md) | [**日本語**](MANUAL.ja.md)

---

## What is AI Assistant?

AI Assistant is a smart chat tool that connects you with AI models. Beyond just chatting, it can control **Blender** (3D modeling) and **GIMP** (image editing) through plugins — you tell the AI what to do, and it operates the software for you.

---

## 1. Chat — Talk to the AI

This is the main screen where you talk to the AI.

![Chat main screen](img/01_chat_overview.png)

### Left Sidebar — Conversation List

All your chats are listed here. Each conversation keeps its own history.

| Button | What it does |
|---|---|
| **+ New** | Start a new conversation |
| **Pencil icon** | Rename the conversation |
| **Trash icon** | Delete the conversation |

### Chat Area (Center)

- Your messages appear on the **right** (blue bubbles)
- AI replies appear on the **left** (gray bubbles)
- AI replies stream in real-time, word by word

### Bottom Toolbar

![Selectors bar](img/05_chat_selectors.png)

- **Left dropdown**: Choose which AI model to use (e.g., DeepSeek, GPT-4o, Claude)
- **Right dropdown**: Choose a "role" — this tells the AI how to behave (e.g., "3D designer", "image editor")
- **Token count**: Shows how many tokens used so far

### How to Use

1. Click **+ New** to start a fresh conversation
2. Type your message in the input box at the bottom
3. Press **Send** or hit Enter
4. Wait for the AI to reply

![Typing a message](img/03_chat_typing.png)

> **Tip**: Hover over any of your own messages to see Copy and Re-input buttons.

---

## 2. AI Bridge — Connect to Blender / GIMP

This page manages connections to external tools. When connected, the AI can control Blender or GIMP based on your instructions.

![AI Bridge overview](img/07_mcp_overview.png)

### What You See

- **Tab bar**: Shows all available services (Blender, GIMP, etc.)
- **Status badge**: Green = Connected, Gray = Disconnected
- **Log area**: Real-time activity log (auto-refreshes every 3 seconds)

![Service detail](img/08_mcp_service_detail.png)

### How to Use

1. Click on a service tab to see its details
2. Click **Connect** to establish the connection
3. Once connected, the status badge turns green
4. Now when you chat, the AI can use that tool
5. Click **Disconnect** when done

> **Tip**: You can connect to multiple services at the same time!

---

## 3. Roles — Set the AI's Personality

Roles are system prompts that tell the AI how to behave. For example, a "Blender Expert" role makes the AI act like a 3D modeling specialist.

![Role Management page](img/09_roles_overview.png)

### Built-in Categories

| Category | Purpose |
|---|---|
| **Custom** | Roles you create yourself |
| **System** | Built-in default roles |
| **Blender** | Roles for 3D modeling tasks |
| **GIMP** | Roles for image editing tasks |

### How to Create a Role

1. Click **+ New Role**
2. Fill in the **Title** (e.g., "Blender Expert")
3. Select a **Category**
4. Write the **Content** — describe how the AI should behave
5. Click **Save**

![New role dialog](img/10_roles_new_dialog.png)

### How to Use a Role

In the Chat page, select the role from the dropdown menu at the bottom. The AI will follow that role's instructions during the conversation.

### Editing Roles

You can edit roles directly in the table — just click the **Edit** button on any row, or click directly on a title/content cell.

> **Tip**: Use category tabs (All / Custom / System / Blender / GIMP) to filter the list.

---

## 4. Prompts — Save Your Favorite Messages

Save messages you send often as templates, so you can reuse them with one click.

![User Prompts page](img/11_prompts_overview.png)

### How to Use

1. Click **+ New Prompt**
2. Give it a **Title** and write the **Content**
3. Click **Save**
4. When you want to use a saved prompt, click **Copy** on it
5. Go to the Chat page and paste it in

---

## 5. Models — Add Your AI Models

You can add models from many different providers. The system comes with built-in presets for popular models.

![Model Settings page](img/12_models_overview.png)

### Supported Providers

DeepSeek, OpenAI (GPT-4o), Anthropic (Claude), Google (Gemini), Moonshot, Alibaba (Qwen), Baidu (ERNIE), Tencent, Groq, Together AI, xAI, OpenRouter, Ollama (local), and more.

### How to Add a Model

1. Click **+ Add Model**
2. Click a **Quick fill** button to auto-fill a popular model
3. Or fill in manually:
   - **Provider**: Who provides the model (e.g., DeepSeek)
   - **Name**: A name you'll recognize
   - **Model**: The model ID (e.g., deepseek-chat)
   - **Base URL**: The API address
   - **API Key**: Your secret key
   - **Proxy URL**: (optional) If you need a proxy
4. Click **Create**

![Adding a model](img/13_models_add_form.png)

### Other Actions

| Button | What it does |
|---|---|
| **Set Active** | Make this model the default for chatting |
| **Test** | Check if the connection works |
| **Edit** | Change the settings |
| **Delete** | Remove the model |

---

## 6. Settings — Configure the System

### Notifications

Set up a webhook (e.g., Feishu/Lark bot) to get notified when the AI finishes a task.

![Notifications settings](img/14_settings_notify.png)

- **Webhook URL**: The address of your notification bot
- **Keywords**: If the AI's reply contains these words, a notification is sent
  - Leave empty to get notified for every task

### Other Settings

![Other settings](img/15_settings_other.png)

- **Blender Working Directory**: Where files created by Blender are saved

---

## 7. Help — In-App Guide

The Help page has three tabs:

| Tab | What you'll find |
|---|---|
| **Blender MCP** | Step-by-step guide to install and connect the Blender plugin |
| **GIMP MCP** | Step-by-step guide to install and connect the GIMP plugin |
| **Agent Help** | How the system works and useful tips |

![Help - Agent tab](img/17_help_agent.png)

---

## Quick Start

1. **Add a model** (page: Models) — click a preset, fill in your API key, click Create, then Set Active
2. **Connect a tool** (page: AI Bridge) — click Connect on Blender or GIMP
3. **Start chatting** (page: Chat) — click + New, type a message, press Send
4. **Set a role** (optional) — pick a role from the dropdown to guide the AI

---

## Navigation

The left sidebar is your main menu. Click any icon to jump to that page.

![Sidebar navigation](img/18_sidebar_nav.png)

| Icon | Page |
|---|---|
| 💬 | Chat |
| 🔌 | AI Bridge |
| 📝 | Roles |
| 📋 | Prompts |
| 🧠 | Models |
| ⚙️ | Settings |
| ❓ | Help |

At the bottom of the sidebar, you can:
- **Toggle dark/light theme** with the sun/moon button
- **Switch language** (EN / 中文 / 日本語) with the globe button
