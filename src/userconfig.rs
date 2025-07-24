use std::fs::create_dir_all;
use std::path;
use std::env::{current_exe};
use windows_registry::{Transaction, CLASSES_ROOT};
use directories::BaseDirs;

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
        create_dir_all(&leek_dir)?;
    }

    Ok(leek_dir)
}