// Ambil inisial (huruf/angka pertama, uppercase) untuk avatar.
export function initial(name) {
  for (const ch of name || "") {
    if (/[\p{L}\p{N}]/u.test(ch)) return ch.toUpperCase();
  }
  return "?";
}
