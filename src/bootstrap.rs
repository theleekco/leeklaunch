use serde::Deserialize;
use reqwest;
use phf::phf_map;
use std::{path, fs, io};
use zip::ZipArchive;
use std::process::{Command};

use crate::{userconfig};

#[allow(dead_code)]
#[derive(Deserialize, Debug)]
pub struct ClientSettingsResponse {
    pub version: String,
    #[serde(rename = "clientVersionUpload")]
    pub client_version_upload: String,
    #[serde(rename = "bootstrapperVersion")]
    pub bootstrapper_version: String,
}

#[allow(dead_code)]
#[derive(Debug)]
pub struct FileInfo {
    pub filename: String,
    pub md5_hash: String,
    pub compressed_size: u64,
    pub uncompressed_size: u64,
}

const APP_SETTINGS: &str = "<Settings>
<ContentFolder>content</ContentFolder>
<BaseUrl>http://www.roblox.com</BaseUrl>
</Settings>";

static EXTRACTION_ROOTS: phf::Map<&'static str, &'static str> = phf_map! {
    "RobloxApp.zip" => "./",
    "redist.zip" => "./",
    "shaders.zip" => "./shaders",
    "ssl.zip" => "./ssl",
    "WebView2.zip" => "./",
    "WebView2RuntimeInstaller.zip" => "./WebView2RuntimeInstaller",
    "content-avatar.zip" => "./content/avatar",
    "content-configs.zip" => "./content/configs",
    "content-fonts.zip" => "./content/fonts",
    "content-sky.zip" => "./content/sky",
    "content-sounds.zip" => "./content/sounds",
    "content-textures2.zip" => "./content/textures",
    "content-models.zip" => "./content/models",
    "content-platform-fonts.zip" => "./PlatformContent/pc/fonts",
    "content-platform-dictionaries.zip" => "./PlatformContent/pc/shared_compression_dictionaries",
    "content-terrain.zip" => "./PlatformContent/pc/terrain",
    "content-textures3.zip" => "./PlatformContent/pc/textures",
    "extracontent-places.zip" => "./ExtraContent/places",
    "extracontent-luapackages.zip" => "./ExtraContent/LuaPackages",
    "extracontent-translations.zip" => "./ExtraContent/translations",
    "extracontent-models.zip" => "./ExtraContent/models",
    "extracontent-textures.zip" => "./ExtraContent/textures",
};

pub async fn get_client_settings() -> Result<ClientSettingsResponse, Box<dyn std::error::Error>> {
    let app_config = userconfig::get_config().await?;

    let channel = app_config.channel;
    let client_settings_url = if channel == "LIVE" {
        "https://clientsettingscdn.roblox.com/v2/client-version/WindowsPlayer".to_string()
    } else {
        format!("https://clientsettingscdn.roblox.com/v2/client-version/WindowsPlayer/channel/{}",
            &channel)
    };
    
    let response = reqwest::get(client_settings_url).await?.error_for_status()?;
    let client_info: ClientSettingsResponse = response.json().await?;

    Ok(client_info)
}

pub async fn get_version_manifest() -> Result<String, Box<dyn std::error::Error>> {
    let app_config = userconfig::get_config().await?;

    let channel = app_config.channel;
    let client_settings = get_client_settings().await?;

    let manifest_url = if channel != "LIVE" {
        format!(
            "https://setup.rbxcdn.com/channel/common/{}-rbxPkgManifest.txt",
            client_settings.client_version_upload
        )
    } else {
        format!(
            "https://setup.rbxcdn.com/{}-rbxPkgManifest.txt",
            client_settings.client_version_upload
        )
    };

    let response = reqwest::get(manifest_url).await?.error_for_status()?;
    let manifest_content = response.text().await?;

    Ok(manifest_content)
}

pub async fn get_archive_manifest() -> Result<Vec<FileInfo>, Box<dyn std::error::Error>> {
    let mut file_list = Vec::new();
    let mut manifest = get_version_manifest().await?;

    if manifest.lines().next() != Some("v0") {
        return Err("Invalid manifest version".into())
    }

    manifest = manifest.lines().skip(1).collect::<Vec<_>>().join("\n");

    let mut lines = manifest.lines();
    loop {
        let filename = match lines.next() {
            Some(line) => line,
            None => break,
        };
        let md5_hash = match lines.next() {
            Some(line) => line,
            None => break,
        };
        let compressed_size = match lines.next() {
            Some(line) => line,
            None => break,
        };
        let uncompressed_size = match lines.next() {
            Some(line) => line,
            None => break,
        };

        file_list.push(FileInfo {
            filename: filename.to_string(),
            md5_hash: md5_hash.to_string(),
            compressed_size: compressed_size.parse()?,
            uncompressed_size: uncompressed_size.parse()?,
        });
    }

    Ok(file_list)
}

pub async fn save_deployment() {
    let client_settings = get_client_settings().await.unwrap();
    let manifest = get_archive_manifest().await;

    let versions_dir = userconfig::get_data_directory().await.unwrap().join("versions");
    let current_version = versions_dir.join(&client_settings.client_version_upload);

    println!("Now installing Player to {}", current_version.display());

    match manifest {
        Ok(files) => {
            for file in files.iter() {
                let archive_url = format!("https://setup.rbxcdn.com/{}-{}", client_settings.client_version_upload, file.filename);
                
                match reqwest::get(&archive_url).await {
                    Ok(resp) => {
                        let bytes = resp.bytes().await.unwrap();
                        let reader = io::Cursor::new(&bytes);

                        let extraction_root = EXTRACTION_ROOTS
                            .get(file.filename.as_str())
                            .map(|root| root.to_string())
                            .unwrap_or_else(|| "./".to_string());

                        let mut archive = match ZipArchive::new(reader) {
                            Ok(archive) => archive,
                            Err(_e) => {
                                // assuming that the file, is infact, not a zip
                                let outpath = current_version.join(&file.filename);

                                if let Err(e) = fs::write(&outpath, &bytes) {
                                    eprintln!("Failed to write file {}: {}", file.filename, e);
                                }
                                continue;
                            }
                        };

                        let extract_path = current_version.join(path::Path::new(&extraction_root));

                        for i in 0..archive.len() {
                            let mut file = archive.by_index(i).unwrap();
                            let outpath = match file.enclosed_name() {
                                Some(path) => extract_path.join(path),
                                None => continue,
                            };

                            if file.is_dir() {
                                fs::create_dir_all(&outpath).unwrap();
                            } else {
                                if let Some(p) = outpath.parent() {
                                    if !p.exists() {
                                        fs::create_dir_all(p).unwrap();
                                    }
                                }

                                let mut outfile = fs::File::create(&outpath).unwrap();
                                io::copy(&mut file, &mut outfile).unwrap();
                            }
                        }
                    },
                    Err(e) => {
                        eprintln!("Failed to download {}: {}", file.filename, e);
                        continue;
                    }
                };
            }
        },
        Err(e) => {
            eprintln!("Error fetching archive manifest: {}", e);
            return;
        }
    }

    let mut app_settings_file = fs::File::create(current_version.join("AppSettings.xml")).unwrap();
    io::copy(&mut APP_SETTINGS.as_bytes(), &mut app_settings_file).unwrap();
}

pub async fn probe_install() -> Result<String, Box<dyn std::error::Error>> {
    let versions_dir = userconfig::get_data_directory().await?.join("versions");
    let client_settings = get_client_settings().await?;

    let player_path = versions_dir.join(format!("{}/RobloxPlayerBeta.exe", client_settings.client_version_upload));

    if !fs::exists(&player_path)? {
        save_deployment().await
    }

    Ok(player_path.to_string_lossy().into_owned())
}

pub async fn launch_player(player_args: Vec<String>) -> Result<(), Box<dyn std::error::Error>> {
    let player_path = probe_install().await?;

    println!("Launching player");

    Command::new(player_path)
        .args(player_args)
        .spawn()
        .map_err(|e| format!("Failed to launch player: {}", e))?;

    Ok(())
}