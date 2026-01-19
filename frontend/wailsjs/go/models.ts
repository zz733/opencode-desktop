export namespace main {
	
	export class AccountSettings {
	    quotaRefreshInterval: number;
	    autoRefreshQuota: boolean;
	    quotaAlertThreshold: number;
	    showQuotaInStatusBar: boolean;
	    defaultLoginMethod: string;
	    preferredOAuthProvider: string;
	    exportEncryption: boolean;
	    autoBackup: boolean;
	    backupRetentionDays: number;
	    autoChangeMachineId: boolean;
	
	    static createFrom(source: any = {}) {
	        return new AccountSettings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.quotaRefreshInterval = source["quotaRefreshInterval"];
	        this.autoRefreshQuota = source["autoRefreshQuota"];
	        this.quotaAlertThreshold = source["quotaAlertThreshold"];
	        this.showQuotaInStatusBar = source["showQuotaInStatusBar"];
	        this.defaultLoginMethod = source["defaultLoginMethod"];
	        this.preferredOAuthProvider = source["preferredOAuthProvider"];
	        this.exportEncryption = source["exportEncryption"];
	        this.autoBackup = source["autoBackup"];
	        this.backupRetentionDays = source["backupRetentionDays"];
	        this.autoChangeMachineId = source["autoChangeMachineId"];
	    }
	}
	export class AntigravityAuthStatus {
	    installed: boolean;
	    version: string;
	    latestVersion: string;
	    updateAvailable: boolean;
	
	    static createFrom(source: any = {}) {
	        return new AntigravityAuthStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.installed = source["installed"];
	        this.version = source["version"];
	        this.latestVersion = source["latestVersion"];
	        this.updateAvailable = source["updateAvailable"];
	    }
	}
	export class LoggingConfig {
	    enabled: boolean;
	    level: string;
	    maxFileSize: number;
	    maxFiles: number;
	    rotateDaily: boolean;
	    logToConsole: boolean;
	
	    static createFrom(source: any = {}) {
	        return new LoggingConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.enabled = source["enabled"];
	        this.level = source["level"];
	        this.maxFileSize = source["maxFileSize"];
	        this.maxFiles = source["maxFiles"];
	        this.rotateDaily = source["rotateDaily"];
	        this.logToConsole = source["logToConsole"];
	    }
	}
	export class StorageConfig {
	    maxBackups: number;
	    autoBackupEnabled: boolean;
	    backupInterval: number;
	    compressBackups: boolean;
	    cleanupOldBackups: boolean;
	    backupRetentionDays: number;
	
	    static createFrom(source: any = {}) {
	        return new StorageConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.maxBackups = source["maxBackups"];
	        this.autoBackupEnabled = source["autoBackupEnabled"];
	        this.backupInterval = source["backupInterval"];
	        this.compressBackups = source["compressBackups"];
	        this.cleanupOldBackups = source["cleanupOldBackups"];
	        this.backupRetentionDays = source["backupRetentionDays"];
	    }
	}
	export class SecurityConfig {
	    encryptionEnabled: boolean;
	    keyDerivationMethod: string;
	    encryptionAlgorithm: string;
	    tokenStorageMethod: string;
	    autoLockTimeout: number;
	    requirePasswordOnStart: boolean;
	
	    static createFrom(source: any = {}) {
	        return new SecurityConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.encryptionEnabled = source["encryptionEnabled"];
	        this.keyDerivationMethod = source["keyDerivationMethod"];
	        this.encryptionAlgorithm = source["encryptionAlgorithm"];
	        this.tokenStorageMethod = source["tokenStorageMethod"];
	        this.autoLockTimeout = source["autoLockTimeout"];
	        this.requirePasswordOnStart = source["requirePasswordOnStart"];
	    }
	}
	export class ConfigPaths {
	    baseDir: string;
	    dataDir: string;
	    configDir: string;
	    logsDir: string;
	    backupDir: string;
	    tempDir: string;
	    accountsFile: string;
	    settingsFile: string;
	    tagsFile: string;
	
	    static createFrom(source: any = {}) {
	        return new ConfigPaths(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.baseDir = source["baseDir"];
	        this.dataDir = source["dataDir"];
	        this.configDir = source["configDir"];
	        this.logsDir = source["logsDir"];
	        this.backupDir = source["backupDir"];
	        this.tempDir = source["tempDir"];
	        this.accountsFile = source["accountsFile"];
	        this.settingsFile = source["settingsFile"];
	        this.tagsFile = source["tagsFile"];
	    }
	}
	export class AppConfig {
	    version: string;
	    appName: string;
	    dataVersion: string;
	    paths: ConfigPaths;
	    security: SecurityConfig;
	    storage: StorageConfig;
	    logging: LoggingConfig;
	    // Go type: time
	    createdAt: any;
	    // Go type: time
	    lastUpdated: any;
	
	    static createFrom(source: any = {}) {
	        return new AppConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.version = source["version"];
	        this.appName = source["appName"];
	        this.dataVersion = source["dataVersion"];
	        this.paths = this.convertValues(source["paths"], ConfigPaths);
	        this.security = this.convertValues(source["security"], SecurityConfig);
	        this.storage = this.convertValues(source["storage"], StorageConfig);
	        this.logging = this.convertValues(source["logging"], LoggingConfig);
	        this.createdAt = this.convertValues(source["createdAt"], null);
	        this.lastUpdated = this.convertValues(source["lastUpdated"], null);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ConfigInfo {
	    model: string;
	
	    static createFrom(source: any = {}) {
	        return new ConfigInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.model = source["model"];
	    }
	}
	export class ConfigModel {
	    id: string;
	    name: string;
	    provider: string;
	    contextLen?: number;
	    outputLen?: number;
	
	    static createFrom(source: any = {}) {
	        return new ConfigModel(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.provider = source["provider"];
	        this.contextLen = source["contextLen"];
	        this.outputLen = source["outputLen"];
	    }
	}
	
	export class FileInfo {
	    name: string;
	    path: string;
	    type: string;
	    size: number;
	    children?: FileInfo[];
	
	    static createFrom(source: any = {}) {
	        return new FileInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.path = source["path"];
	        this.type = source["type"];
	        this.size = source["size"];
	        this.children = this.convertValues(source["children"], FileInfo);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class GitChange {
	    path: string;
	    status: string;
	    staged: boolean;
	
	    static createFrom(source: any = {}) {
	        return new GitChange(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.status = source["status"];
	        this.staged = source["staged"];
	    }
	}
	export class GitStatus {
	    branch: string;
	    changes: GitChange[];
	    hasRepo: boolean;
	
	    static createFrom(source: any = {}) {
	        return new GitStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.branch = source["branch"];
	        this.changes = this.convertValues(source["changes"], GitChange);
	        this.hasRepo = source["hasRepo"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ImageData {
	    name: string;
	    type: string;
	    data: string;
	
	    static createFrom(source: any = {}) {
	        return new ImageData(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.type = source["type"];
	        this.data = source["data"];
	    }
	}
	export class QuotaDetail {
	    used: number;
	    total: number;
	
	    static createFrom(source: any = {}) {
	        return new QuotaDetail(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.used = source["used"];
	        this.total = source["total"];
	    }
	}
	export class QuotaInfo {
	    main: QuotaDetail;
	    trial: QuotaDetail;
	    reward: QuotaDetail;
	
	    static createFrom(source: any = {}) {
	        return new QuotaInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.main = this.convertValues(source["main"], QuotaDetail);
	        this.trial = this.convertValues(source["trial"], QuotaDetail);
	        this.reward = this.convertValues(source["reward"], QuotaDetail);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class KiroAccount {
	    id: string;
	    email: string;
	    displayName: string;
	    avatar?: string;
	    // Go type: time
	    tokenExpiry: any;
	    loginMethod: string;
	    provider?: string;
	    subscriptionType: string;
	    quota: QuotaInfo;
	    tags: string[];
	    notes?: string;
	    isActive: boolean;
	    // Go type: time
	    lastUsed: any;
	    // Go type: time
	    createdAt: any;
	    machineId?: string;
	    sqmId?: string;
	    devDeviceId?: string;
	
	    static createFrom(source: any = {}) {
	        return new KiroAccount(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.email = source["email"];
	        this.displayName = source["displayName"];
	        this.avatar = source["avatar"];
	        this.tokenExpiry = this.convertValues(source["tokenExpiry"], null);
	        this.loginMethod = source["loginMethod"];
	        this.provider = source["provider"];
	        this.subscriptionType = source["subscriptionType"];
	        this.quota = this.convertValues(source["quota"], QuotaInfo);
	        this.tags = source["tags"];
	        this.notes = source["notes"];
	        this.isActive = source["isActive"];
	        this.lastUsed = this.convertValues(source["lastUsed"], null);
	        this.createdAt = this.convertValues(source["createdAt"], null);
	        this.machineId = source["machineId"];
	        this.sqmId = source["sqmId"];
	        this.devDeviceId = source["devDeviceId"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class KiroAuthStatus {
	    installed: boolean;
	    version: string;
	    latestVersion: string;
	    updateAvailable: boolean;
	
	    static createFrom(source: any = {}) {
	        return new KiroAuthStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.installed = source["installed"];
	        this.version = source["version"];
	        this.latestVersion = source["latestVersion"];
	        this.updateAvailable = source["updateAvailable"];
	    }
	}
	
	export class MCPServer {
	    type: string;
	    command?: string[];
	    url?: string;
	    enabled: boolean;
	    environment?: Record<string, string>;
	    timeout?: number;
	
	    static createFrom(source: any = {}) {
	        return new MCPServer(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.type = source["type"];
	        this.command = source["command"];
	        this.url = source["url"];
	        this.enabled = source["enabled"];
	        this.environment = source["environment"];
	        this.timeout = source["timeout"];
	    }
	}
	export class MCPConfig {
	    mcp: Record<string, MCPServer>;
	
	    static createFrom(source: any = {}) {
	        return new MCPConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.mcp = this.convertValues(source["mcp"], MCPServer, true);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class MCPMarketItem {
	    name: string;
	    description: string;
	    command: string[];
	    envVars?: string[];
	    category: string;
	    docsUrl?: string;
	    configTips?: string;
	
	    static createFrom(source: any = {}) {
	        return new MCPMarketItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.description = source["description"];
	        this.command = source["command"];
	        this.envVars = source["envVars"];
	        this.category = source["category"];
	        this.docsUrl = source["docsUrl"];
	        this.configTips = source["configTips"];
	    }
	}
	
	export class MCPTool {
	    id: string;
	    description: string;
	    parameters: any;
	
	    static createFrom(source: any = {}) {
	        return new MCPTool(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.description = source["description"];
	        this.parameters = source["parameters"];
	    }
	}
	export class Message {
	    role: string;
	    content: string;
	
	    static createFrom(source: any = {}) {
	        return new Message(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.role = source["role"];
	        this.content = source["content"];
	    }
	}
	export class OhMyOpenCodeStatus {
	    installed: boolean;
	    version: string;
	
	    static createFrom(source: any = {}) {
	        return new OhMyOpenCodeStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.installed = source["installed"];
	        this.version = source["version"];
	    }
	}
	export class OpenCodeStatus {
	    installed: boolean;
	    running: boolean;
	    connected: boolean;
	    path: string;
	    version: string;
	    port: number;
	    workDir: string;
	
	    static createFrom(source: any = {}) {
	        return new OpenCodeStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.installed = source["installed"];
	        this.running = source["running"];
	        this.connected = source["connected"];
	        this.path = source["path"];
	        this.version = source["version"];
	        this.port = source["port"];
	        this.workDir = source["workDir"];
	    }
	}
	export class Provider {
	    id: string;
	    name: string;
	
	    static createFrom(source: any = {}) {
	        return new Provider(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	    }
	}
	export class ProviderInfo {
	    all: Provider[];
	    connected: string[];
	    default: Record<string, string>;
	
	    static createFrom(source: any = {}) {
	        return new ProviderInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.all = this.convertValues(source["all"], Provider);
	        this.connected = source["connected"];
	        this.default = source["default"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class QuotaAlert {
	    accountId: string;
	    accountName: string;
	    quotaType: string;
	    usage: number;
	    message: string;
	
	    static createFrom(source: any = {}) {
	        return new QuotaAlert(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.accountId = source["accountId"];
	        this.accountName = source["accountName"];
	        this.quotaType = source["quotaType"];
	        this.usage = source["usage"];
	        this.message = source["message"];
	    }
	}
	
	
	export class SearchResult {
	    path: string;
	    line: number;
	    content: string;
	    match: string;
	
	    static createFrom(source: any = {}) {
	        return new SearchResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.line = source["line"];
	        this.content = source["content"];
	        this.match = source["match"];
	    }
	}
	
	export class Session {
	    id: string;
	    title: string;
	
	    static createFrom(source: any = {}) {
	        return new Session(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.title = source["title"];
	    }
	}
	
	export class Tag {
	    name: string;
	    color: string;
	    description?: string;
	
	    static createFrom(source: any = {}) {
	        return new Tag(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.color = source["color"];
	        this.description = source["description"];
	    }
	}
	export class TokenInfo {
	    access_token: string;
	    refresh_token: string;
	    // Go type: time
	    expires_at: any;
	    token_type: string;
	
	    static createFrom(source: any = {}) {
	        return new TokenInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.access_token = source["access_token"];
	        this.refresh_token = source["refresh_token"];
	        this.expires_at = this.convertValues(source["expires_at"], null);
	        this.token_type = source["token_type"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class UIUXProMaxStatus {
	    installed: boolean;
	    version: string;
	    latestVersion: string;
	    updateAvailable: boolean;
	
	    static createFrom(source: any = {}) {
	        return new UIUXProMaxStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.installed = source["installed"];
	        this.version = source["version"];
	        this.latestVersion = source["latestVersion"];
	        this.updateAvailable = source["updateAvailable"];
	    }
	}

}

