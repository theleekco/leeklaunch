use winresource::{WindowsResource};

fn main() {
    println!("cargo:rerun-if-changed=build.rs"); 

    if std::env::var("CARGO_CFG_TARGET_OS").unwrap() == "windows" {
        WindowsResource::new()
            .set("FileDescription", "leeklaunch")
            .set("ProductName", "leeklaunch")
            .compile()
            .unwrap();
    }
}