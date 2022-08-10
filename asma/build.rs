use build_data::get_env;

pub fn main() {
    build_data::set_SOURCE_DATE();
    build_data::set_SOURCE_TIME();
    build_data::set_SOURCE_TIMESTAMP();
    build_data::set_SOURCE_EPOCH_TIME();
    build_data::set_BUILD_DATE();
    build_data::set_BUILD_TIME();
    build_data::set_BUILD_TIMESTAMP();
    build_data::set_BUILD_EPOCH_TIME();
    build_data::set_BUILD_HOSTNAME();
    build_data::set_GIT_BRANCH();
    build_data::set_GIT_COMMIT();
    build_data::set_GIT_COMMIT_SHORT();
    build_data::set_GIT_DIRTY();
    build_data::set_RUSTC_VERSION();
    build_data::set_RUSTC_VERSION_SEMVER();
    build_data::set_RUST_CHANNEL();
    let shared_key_salt = get_env("SHARED_KEY_SALT").unwrap();
    if !shared_key_salt.is_none() {
        println!(
            "cargo:rustc-env=SHARED_KEY_SALT={}",
            shared_key_salt.unwrap().to_string()
        );
    }
}
