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
}

#[derive(Serialize, Deserialize, Clone)]
pub struct MemoRule {
    pub title: String,
    pub update_rule: String,
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
        })
    } else {
        Config {
            api_key: String::new(),
            model_id: String::new(),
            base_url: String::new(),
            reasoning_enabled: false,
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
) -> bool {
    let path = get_config_path(&app);
    let config = Config {
        api_key,
        model_id,
        base_url,
        reasoning_enabled,
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

/// Import from MemoChat rules JSON format
#[tauri::command]
fn import_from_memochat(json: String) -> Result<RulePack, String> {
    let val: Value = serde_json::from_str(&json).map_err(|e| e.to_string())?;
    let system_prompt = val["systemPrompt"].as_str().unwrap_or("").to_string();
    let rules = val["rules"]
        .as_array()
        .map(|arr| {
            arr.iter()
                .filter_map(|r| {
                    Some(MemoRule {
                        title: r["title"].as_str()?.to_string(),
                        update_rule: r["updateRule"].as_str()?.to_string(),
                    })
                })
                .collect()
        })
        .unwrap_or_default();

    let now = chrono::Local::now().format("%Y-%m-%dT%H:%M:%S").to_string();
    Ok(RulePack {
        id: format!("imported_{}", chrono::Local::now().timestamp_millis()),
        name: "Imported from MemoChat".to_string(),
        description: String::new(),
        author: String::new(),
        version: "1.0.0".to_string(),
        system_prompt,
        rules,
        tags: vec!["imported".to_string()],
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
