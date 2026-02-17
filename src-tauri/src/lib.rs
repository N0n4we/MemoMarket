use serde::{Deserialize, Serialize};
use serde_json::Value;
use std::fs;
use std::path::PathBuf;
use tauri::{AppHandle, Manager};

#[derive(Serialize, Deserialize, Clone)]
struct Config {
    api_key: String,
    model_id: String,
    base_url: String,
    #[serde(default)]
    reasoning_enabled: bool,
    #[serde(default)]
    channels_json: String,
}

#[derive(Serialize, Deserialize, Clone)]
pub struct MemoRule {
    pub title: String,
    pub update_rule: String,
}

#[derive(Serialize, Deserialize, Clone)]
pub struct Memo {
    pub title: String,
    pub content: String,
}

#[derive(Serialize, Deserialize, Clone)]
pub struct RulePack {
    pub id: String,
    pub name: String,
    pub description: String,
    pub author: String,
    pub version: String,
    pub system_prompt: String,
    pub rules: Vec<MemoRule>,
    #[serde(default)]
    pub memos: Vec<Memo>,
    #[serde(default)]
    pub tags: Vec<String>,
    #[serde(default)]
    pub created_at: String,
    #[serde(default)]
    pub updated_at: String,
}

fn get_config_path(app: &AppHandle) -> PathBuf {
    let config_dir = app
        .path()
        .app_config_dir()
        .expect("failed to get config dir");
    fs::create_dir_all(&config_dir).ok();
    config_dir.join("config.json")
}

fn get_packs_dir(app: &AppHandle) -> PathBuf {
    let config_dir = app
        .path()
        .app_config_dir()
        .expect("failed to get config dir");
    let packs = config_dir.join("packs");
    fs::create_dir_all(&packs).ok();
    packs
}

fn get_installed_path(app: &AppHandle) -> PathBuf {
    let config_dir = app
        .path()
        .app_config_dir()
        .expect("failed to get config dir");
    fs::create_dir_all(&config_dir).ok();
    config_dir.join("installed.json")
}

#[tauri::command]
fn load_config(app: AppHandle) -> Config {
    let path = get_config_path(&app);
    if path.exists() {
        let json = fs::read_to_string(&path).unwrap_or_default();
        serde_json::from_str(&json).unwrap_or(Config {
            api_key: String::new(),
            model_id: String::new(),
            base_url: String::new(),
            reasoning_enabled: false,
            channels_json: String::new(),
        })
    } else {
        Config {
            api_key: String::new(),
            model_id: String::new(),
            base_url: String::new(),
            reasoning_enabled: false,
            channels_json: String::new(),
        }
    }
}

#[tauri::command]
fn save_config(
    app: AppHandle,
    api_key: String,
    model_id: String,
    base_url: String,
    reasoning_enabled: bool,
    channels_json: String,
) -> bool {
    let path = get_config_path(&app);
    let config = Config {
        api_key,
        model_id,
        base_url,
        reasoning_enabled,
        channels_json,
    };
    let json = serde_json::to_string_pretty(&config).unwrap();
    fs::write(path, json).is_ok()
}

#[tauri::command]
fn load_packs(app: AppHandle) -> Vec<RulePack> {
    let dir = get_packs_dir(&app);
    let mut packs = Vec::new();
    if let Ok(entries) = fs::read_dir(&dir) {
        for entry in entries.flatten() {
            if entry.path().extension().map_or(false, |e| e == "json") {
                if let Ok(json) = fs::read_to_string(entry.path()) {
                    if let Ok(pack) = serde_json::from_str::<RulePack>(&json) {
                        packs.push(pack);
                    }
                }
            }
        }
    }
    packs.sort_by(|a, b| b.updated_at.cmp(&a.updated_at));
    packs
}

#[tauri::command]
fn save_pack(app: AppHandle, pack: RulePack) -> bool {
    let dir = get_packs_dir(&app);
    let path = dir.join(format!("{}.json", pack.id));
    let json = serde_json::to_string_pretty(&pack).unwrap();
    fs::write(path, json).is_ok()
}

#[tauri::command]
fn delete_pack(app: AppHandle, id: String) -> bool {
    let dir = get_packs_dir(&app);
    let path = dir.join(format!("{}.json", id));
    if path.exists() {
        fs::remove_file(path).is_ok()
    } else {
        false
    }
}

#[tauri::command]
fn export_pack(pack: RulePack) -> String {
    serde_json::to_string_pretty(&pack).unwrap_or_default()
}

#[tauri::command]
fn import_pack_json(json: String) -> Result<RulePack, String> {
    serde_json::from_str(&json).map_err(|e| e.to_string())
}

#[tauri::command]
fn load_installed(app: AppHandle) -> Vec<String> {
    let path = get_installed_path(&app);
    if path.exists() {
        let json = fs::read_to_string(&path).unwrap_or_default();
        serde_json::from_str(&json).unwrap_or_default()
    } else {
        Vec::new()
    }
}

#[tauri::command]
fn save_installed(app: AppHandle, ids: Vec<String>) -> bool {
    let path = get_installed_path(&app);
    let json = serde_json::to_string_pretty(&ids).unwrap();
    fs::write(path, json).is_ok()
}

/// Export a pack in MemoChat-compatible format (for importing into MemoChat)
#[tauri::command]
fn export_for_memochat(pack: RulePack) -> String {
    let memochat_format = serde_json::json!({
        "systemPrompt": pack.system_prompt,
        "rules": pack.rules.iter().map(|r| {
            serde_json::json!({
                "title": r.title,
                "updateRule": r.update_rule,
            })
        }).collect::<Vec<_>>(),
    });
    serde_json::to_string_pretty(&memochat_format).unwrap_or_default()
}

/// Import from MemoChat memo-pack.json file
#[tauri::command]
fn import_from_memochat(app: AppHandle) -> Result<RulePack, String> {
    // Get MemoChat config directory
    let memochat_config_dir = app
        .path()
        .app_config_dir()
        .map_err(|e| e.to_string())?
        .parent()
        .ok_or("Failed to get parent directory")?
        .join("com.memochat.app");

    let memo_pack_path = memochat_config_dir.join("memo-pack.json");

    if !memo_pack_path.exists() {
        return Err("MemoChat memo-pack.json not found. Please make sure MemoChat is installed and has been run at least once.".to_string());
    }

    let json = fs::read_to_string(&memo_pack_path)
        .map_err(|e| format!("Failed to read memo-pack.json: {}", e))?;

    let val: Value = serde_json::from_str(&json)
        .map_err(|e| format!("Failed to parse memo-pack.json: {}", e))?;

    let rules = val["rules"]
        .as_array()
        .map(|arr| {
            arr.iter()
                .filter_map(|r| {
                    Some(MemoRule {
                        title: r["description"].as_str()?.to_string(),
                        update_rule: r["update_rule"].as_str()?.to_string(),
                    })
                })
                .collect()
        })
        .unwrap_or_default();

    let memos = val["memos"]
        .as_array()
        .map(|arr| {
            arr.iter()
                .filter_map(|m| {
                    Some(Memo {
                        title: m["title"].as_str()?.to_string(),
                        content: m["content"].as_str()?.to_string(),
                    })
                })
                .collect()
        })
        .unwrap_or_default();

    let now = chrono::Local::now().format("%Y-%m-%dT%H:%M:%S").to_string();
    Ok(RulePack {
        id: format!("imported_{}", chrono::Local::now().timestamp_millis()),
        name: "Imported from MemoChat".to_string(),
        description: "Current memo pack from MemoChat".to_string(),
        author: String::new(),
        version: "1.0.0".to_string(),
        system_prompt: String::new(),
        rules,
        memos,
        tags: vec!["imported".to_string(), "memochat".to_string()],
        created_at: now.clone(),
        updated_at: now,
    })
}

#[cfg_attr(mobile, tauri::mobile_entry_point)]
pub fn run() {
    tauri::Builder::default()
        .invoke_handler(tauri::generate_handler![
            load_config,
            save_config,
            load_packs,
            save_pack,
            delete_pack,
            export_pack,
            import_pack_json,
            load_installed,
            save_installed,
            export_for_memochat,
            import_from_memochat,
        ])
        .run(tauri::generate_context!())
        .expect("error while running tauri application");
}
