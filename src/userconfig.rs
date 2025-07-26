use std::fs;
use std::path;
use std::env::{current_exe};
use windows_registry::{Transaction, CLASSES_ROOT};
use directories::BaseDirs;
use config::Config;
use std::collections::HashMap;
use serde;
use serde::{Deserialize, Serialize};

#[derive(Debug, Deserialize, Serialize, Clone)]
pub struct AppConfig {
    #[serde(default)]
    pub channel: String,
    #[serde(default = "HashMap::new")]
    pub fflags: HashMap<String, bool>,
}

impl Default for AppConfig {
    fn default() -> Self {
        AppConfig {
            channel: "LIVE".to_string(),
            fflags: HashMap::new(),
        }
    }
}
pub fn register_proto() -> Result<(), Box<dyn std::error::Error>> {
    const PATHS: [&str; 2] = [
        "roblox",
        "roblox-player",
    ];
    let self_exe = current_exe()?.to_string_lossy().into_owned();
    
    // would be better to do this outside a loop
    // but porting go over is more fun
    for path in PATHS {
        let key_path = format!("{}\\shell\\open\\command", path);
        let tx = Transaction::new()?;

        let key = CLASSES_ROOT
            .options()
            .read()
            .write()
            .create()
            .transaction(&tx)
            .open(key_path)?;

        let reg_value = format!("\"{}\" --player \"%1\"", &self_exe);

        // borrow checker pissed me off here
        if &key.get_string("").unwrap_or_default() != &reg_value {
            key.set_string("", &reg_value)?;
        }

        tx.commit()?;
    }

    Ok(())
}

pub async fn get_data_directory() -> Result<path::PathBuf, Box<dyn std::error::Error>> {
    let base_dirs = BaseDirs::new().ok_or("base directory bruhed")?;
    let data_dir = base_dirs
        .data_local_dir();

    let leek_dir = data_dir.join("leeklaunch");

    if !leek_dir.exists() {
        fs::create_dir_all(&leek_dir)?;
    }

    Ok(leek_dir)
}

pub async fn write_config(cfg: &AppConfig) -> Result<bool, Box<dyn std::error::Error>> {
    let data_dir = get_data_directory().await?;
    let cfg_path = data_dir.join("leek.json");

    let json_string = serde_json::to_string_pretty(cfg)?;
    fs::write(&cfg_path, &json_string)?;

    Ok(true)
}

pub async fn get_config() -> Result<AppConfig, Box<dyn std::error::Error>> {
    let data_dir = get_data_directory().await?;
    let cfg_path = data_dir.join("leek.json");

    if !cfg_path.exists() {
        let default_config = AppConfig::default();
        let json_string = serde_json::to_string_pretty(&default_config)?;

        fs::write(&cfg_path, &json_string)?;
    }

    let cfg = Config::builder()
        .add_source(config::File::from(cfg_path.to_owned()))
        .build()?
        .try_deserialize::<AppConfig>()?;

    // make it so missing values are set to default
    // does this count as a hack?
    let cfg_str = serde_json::to_string_pretty(&cfg)?;
    fs::write(cfg_path, cfg_str)?;
        
    Ok(cfg)
}