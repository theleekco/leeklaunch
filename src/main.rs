use clap::Parser;

mod userconfig;
mod bootstrap;

#[derive(Parser, Debug)]
#[command(version, about, long_about = None)]
struct Args {
    // Deeplink
    #[arg(long)]
    player: Option<String>,

    // Reinstall the current installation
    #[arg(long)]
    reinstall: Option<bool>,
}

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    let args = Args::parse();

    println!("Hello, world!");
    
    userconfig::register_proto().unwrap();

    if let Some(reinstall) = args.reinstall {
        if reinstall {
            bootstrap::save_deployment().await;
            println!("Reinstallation complete");
        }
    }

    if let Some(player_args) = args.player {
        bootstrap::launch_player(vec![player_args]).await?;
    };

    Ok(())
}
