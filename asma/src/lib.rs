extern crate web_sys;
use aes::Aes256;
use block_modes::block_padding::Pkcs7;
use block_modes::{BlockMode, Cbc};
use md5::compute as md5_digest;
use std::collections::HashMap;
use wasm_bindgen::prelude::*;

type Aes256Cbc = Cbc<Aes256, Pkcs7>;

// macro_rules! log {
//     ($($t:tt)*) => {
//         web_sys::console::log_1(&format!($($t)*).into());
//     }
// }

#[wasm_bindgen]
pub fn build_info() -> JsValue {
    let mut hm = HashMap::new();
    hm.insert("GIT_BRANCH", env!("GIT_BRANCH"));
    hm.insert("SOURCE_DATE", env!("SOURCE_DATE"));
    hm.insert("SOURCE_TIME", env!("SOURCE_TIME"));
    hm.insert("SOURCE_TIMESTAMP", env!("SOURCE_TIMESTAMP"));
    hm.insert("SOURCE_EPOCH_TIME", env!("SOURCE_EPOCH_TIME"));
    hm.insert("BUILD_DATE", env!("BUILD_DATE"));
    hm.insert("BUILD_TIME", env!("BUILD_TIME"));
    hm.insert("BUILD_TIMESTAMP", env!("BUILD_TIMESTAMP"));
    hm.insert("BUILD_EPOCH_TIME", env!("BUILD_EPOCH_TIME"));
    hm.insert("BUILD_HOSTNAME", env!("BUILD_HOSTNAME"));
    hm.insert("GIT_BRANCH", env!("GIT_BRANCH"));
    hm.insert("GIT_COMMIT", env!("GIT_COMMIT"));
    hm.insert("GIT_COMMIT_SHORT", env!("GIT_COMMIT_SHORT"));
    hm.insert("GIT_DIRTY", env!("GIT_DIRTY"));
    hm.insert("RUSTC_VERSION", env!("RUSTC_VERSION"));
    hm.insert("RUSTC_VERSION_SEMVER", env!("RUSTC_VERSION_SEMVER"));
    hm.insert("RUST_CHANNEL", env!("RUST_CHANNEL"));
    return serde_wasm_bindgen::to_value(&hm).unwrap();
}

#[wasm_bindgen]
pub fn proxy(payload_with_iv: Box<[u8]>, jwt_token: &str) -> Result<String, JsError> {
    // Improve by using SHA 256 bytes only
    let mut key = jwt_token.to_owned();
    let shared_key_salt = option_env!("SHARED_KEY_SALT");
    if !shared_key_salt.is_none() {
        key.push_str(shared_key_salt.unwrap())
    }
    let final_key = format!("{:x}", md5_digest(key));

    if payload_with_iv.len() < 16 {
        return Err(JsError::new("Invalid payload length"));
    }

    let iv = payload_with_iv[..16].to_vec();
    let data = payload_with_iv[16..].to_vec();

    let decrypted = decrypt_aes256(&final_key.as_bytes(), &iv, &data);
    return Ok(String::from_utf8(decrypted).unwrap());
}

fn decrypt_aes256(key: &[u8], iv: &[u8], data: &[u8]) -> Vec<u8> {
    let mut encrypted_data = data.clone().to_owned();
    let cipher = Aes256Cbc::new_from_slices(&key, &iv).unwrap();
    cipher.decrypt(&mut encrypted_data).unwrap().to_vec()
}
