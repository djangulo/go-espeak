function update(id, value) {
  document.getElementById(id).innerHTML = value;
}
function setDefaults() {
  document.getElementById("punctuation-none").checked = true;
  document.getElementById("capitals-none").checked = true;
  document.getElementById("punctuation-list").value = "";
  document.getElementById("rate").value = 175;
  update("rate-value", 175);
  document.getElementById("volume").value = 100;
  update("volume-value", 100);
  document.getElementById("pitch").value = 50;
  update("pitch-value", 50);
  document.getElementById("range").value = 50;
  update("range-value", 50);
  document.getElementById("word-gap").value = 10;
  update("word-gap-value", 10);
}
function togglePunctList() {
  var plist = document.getElementById("punctuation-list");
   if (!document.getElementById("punctuation-some").checked) {
     plist.style.visibility = 'hidden';
     plist.style.display = 'none';
   } else {
    plist.style.visibility = 'visible';
    plist.style.display = 'inline-block';
   }
}