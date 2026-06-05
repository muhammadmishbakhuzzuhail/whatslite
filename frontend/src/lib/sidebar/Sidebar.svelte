<script>
  import { railView } from "../../stores.js";
  import ChatsPane from "./ChatsPane.svelte";
  import SettingsPane from "./SettingsPane.svelte";
  import ProfilePane from "./ProfilePane.svelte";
  import StarredPane from "./StarredPane.svelte";
  import ArchivedPane from "./ArchivedPane.svelte";
  import PrivacyPane from "./PrivacyPane.svelte";
  import StatusPane from "./StatusPane.svelte";
  import ChannelsPane from "./ChannelsPane.svelte";
  import CommunitiesPane from "./CommunitiesPane.svelte";
  import PlaceholderPane from "./PlaceholderPane.svelte";
</script>

<section class="sidebar">
  {#if $railView === "settings"}
    <SettingsPane />
  {:else if $railView === "profile"}
    <ProfilePane />
  {:else if $railView === "starred"}
    <StarredPane />
  {:else if $railView === "archived"}
    <ArchivedPane />
  {:else if $railView === "privacy"}
    <PrivacyPane />
  {:else if $railView === "status"}
    <StatusPane />
  {:else if $railView === "channels"}
    <ChannelsPane />
  {:else if $railView === "communities"}
    <CommunitiesPane />
  {:else}
    <ChatsPane />
  {/if}
</section>
