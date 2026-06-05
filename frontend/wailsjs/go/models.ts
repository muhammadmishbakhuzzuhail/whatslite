export namespace app {
	
	export class ChannelDTO {
	    jid: string;
	    name: string;
	    description: string;
	    subscribers: number;
	    picture: string;
	    verified: boolean;
	    muted: boolean;
	    role: string;
	
	    static createFrom(source: any = {}) {
	        return new ChannelDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.jid = source["jid"];
	        this.name = source["name"];
	        this.description = source["description"];
	        this.subscribers = source["subscribers"];
	        this.picture = source["picture"];
	        this.verified = source["verified"];
	        this.muted = source["muted"];
	        this.role = source["role"];
	    }
	}
	export class ChannelMsgDTO {
	    id: string;
	    serverId: number;
	    type: string;
	    text: string;
	    thumb: string;
	    time: string;
	    views: number;
	
	    static createFrom(source: any = {}) {
	        return new ChannelMsgDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.serverId = source["serverId"];
	        this.type = source["type"];
	        this.text = source["text"];
	        this.thumb = source["thumb"];
	        this.time = source["time"];
	        this.views = source["views"];
	    }
	}
	export class ChatDTO {
	    id: string;
	    name: string;
	    preview: string;
	    time: string;
	    ts: number;
	    group: boolean;
	    sent: boolean;
	    unread: boolean;
	    badge: number;
	    pinned: boolean;
	    muted: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ChatDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.preview = source["preview"];
	        this.time = source["time"];
	        this.ts = source["ts"];
	        this.group = source["group"];
	        this.sent = source["sent"];
	        this.unread = source["unread"];
	        this.badge = source["badge"];
	        this.pinned = source["pinned"];
	        this.muted = source["muted"];
	    }
	}
	export class CommunitySubDTO {
	    jid: string;
	    name: string;
	    isDefault: boolean;
	
	    static createFrom(source: any = {}) {
	        return new CommunitySubDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.jid = source["jid"];
	        this.name = source["name"];
	        this.isDefault = source["isDefault"];
	    }
	}
	export class CommunityDTO {
	    jid: string;
	    name: string;
	    description: string;
	    groups: CommunitySubDTO[];
	
	    static createFrom(source: any = {}) {
	        return new CommunityDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.jid = source["jid"];
	        this.name = source["name"];
	        this.description = source["description"];
	        this.groups = this.convertValues(source["groups"], CommunitySubDTO);
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
	
	export class ContactProfileDTO {
	    jid: string;
	    name: string;
	    phone: string;
	    about: string;
	    saved: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ContactProfileDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.jid = source["jid"];
	        this.name = source["name"];
	        this.phone = source["phone"];
	        this.about = source["about"];
	        this.saved = source["saved"];
	    }
	}
	export class ContactRowDTO {
	    jid: string;
	    name: string;
	    phone: string;
	    saved: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ContactRowDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.jid = source["jid"];
	        this.name = source["name"];
	        this.phone = source["phone"];
	        this.saved = source["saved"];
	    }
	}
	export class GifDTO {
	    id: string;
	    preview: string;
	    mp4: string;
	
	    static createFrom(source: any = {}) {
	        return new GifDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.preview = source["preview"];
	        this.mp4 = source["mp4"];
	    }
	}
	export class GroupMemberDTO {
	    jid: string;
	    name: string;
	    isAdmin: boolean;
	
	    static createFrom(source: any = {}) {
	        return new GroupMemberDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.jid = source["jid"];
	        this.name = source["name"];
	        this.isAdmin = source["isAdmin"];
	    }
	}
	export class GroupInfoDTO {
	    jid: string;
	    name: string;
	    topic: string;
	    owner: string;
	    amAdmin: boolean;
	    participants: GroupMemberDTO[];
	
	    static createFrom(source: any = {}) {
	        return new GroupInfoDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.jid = source["jid"];
	        this.name = source["name"];
	        this.topic = source["topic"];
	        this.owner = source["owner"];
	        this.amAdmin = source["amAdmin"];
	        this.participants = this.convertValues(source["participants"], GroupMemberDTO);
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
	
	export class LinkPreviewDTO {
	    url: string;
	    title: string;
	    desc: string;
	    image: string;
	
	    static createFrom(source: any = {}) {
	        return new LinkPreviewDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.url = source["url"];
	        this.title = source["title"];
	        this.desc = source["desc"];
	        this.image = source["image"];
	    }
	}
	export class MentionDTO {
	    num: string;
	    name: string;
	    jid: string;
	
	    static createFrom(source: any = {}) {
	        return new MentionDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.num = source["num"];
	        this.name = source["name"];
	        this.jid = source["jid"];
	    }
	}
	export class ReactionDTO {
	    emoji: string;
	    count: number;
	    mine: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ReactionDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.emoji = source["emoji"];
	        this.count = source["count"];
	        this.mine = source["mine"];
	    }
	}
	export class MessageDTO {
	    id: string;
	    dir: string;
	    type: string;
	    text: string;
	    thumb: string;
	    time: string;
	    sender: string;
	    senderId: string;
	    senderPhone: string;
	    senderSaved: boolean;
	    status: string;
	    pinned: boolean;
	    edited: boolean;
	    ts: number;
	    quoteId: string;
	    quoteName: string;
	    quoteText: string;
	    mentions: MentionDTO[];
	    reactions: ReactionDTO[];
	
	    static createFrom(source: any = {}) {
	        return new MessageDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.dir = source["dir"];
	        this.type = source["type"];
	        this.text = source["text"];
	        this.thumb = source["thumb"];
	        this.time = source["time"];
	        this.sender = source["sender"];
	        this.senderId = source["senderId"];
	        this.senderPhone = source["senderPhone"];
	        this.senderSaved = source["senderSaved"];
	        this.status = source["status"];
	        this.pinned = source["pinned"];
	        this.edited = source["edited"];
	        this.ts = source["ts"];
	        this.quoteId = source["quoteId"];
	        this.quoteName = source["quoteName"];
	        this.quoteText = source["quoteText"];
	        this.mentions = this.convertValues(source["mentions"], MentionDTO);
	        this.reactions = this.convertValues(source["reactions"], ReactionDTO);
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
	export class ReceiptDTO {
	    name: string;
	    time: string;
	
	    static createFrom(source: any = {}) {
	        return new ReceiptDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.time = source["time"];
	    }
	}
	export class MsgInfoDTO {
	    id: string;
	    type: string;
	    status: string;
	    fromMe: boolean;
	    sent: string;
	    sender: string;
	    senderId: string;
	    readBy: ReceiptDTO[];
	    deliveredTo: ReceiptDTO[];
	
	    static createFrom(source: any = {}) {
	        return new MsgInfoDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.type = source["type"];
	        this.status = source["status"];
	        this.fromMe = source["fromMe"];
	        this.sent = source["sent"];
	        this.sender = source["sender"];
	        this.senderId = source["senderId"];
	        this.readBy = this.convertValues(source["readBy"], ReceiptDTO);
	        this.deliveredTo = this.convertValues(source["deliveredTo"], ReceiptDTO);
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
	export class PollVotesDTO {
	    counts: Record<string, number>;
	    total: number;
	
	    static createFrom(source: any = {}) {
	        return new PollVotesDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.counts = source["counts"];
	        this.total = source["total"];
	    }
	}
	export class ProfileDTO {
	    name: string;
	    phone: string;
	    about: string;
	
	    static createFrom(source: any = {}) {
	        return new ProfileDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.phone = source["phone"];
	        this.about = source["about"];
	    }
	}
	
	
	export class SearchHitDTO {
	    chatJid: string;
	    chatName: string;
	    msgId: string;
	    text: string;
	    time: string;
	    group: boolean;
	
	    static createFrom(source: any = {}) {
	        return new SearchHitDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.chatJid = source["chatJid"];
	        this.chatName = source["chatName"];
	        this.msgId = source["msgId"];
	        this.text = source["text"];
	        this.time = source["time"];
	        this.group = source["group"];
	    }
	}
	export class StatusItemDTO {
	    id: string;
	    type: string;
	    text: string;
	    thumb: string;
	    time: string;
	    ts: number;
	
	    static createFrom(source: any = {}) {
	        return new StatusItemDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.type = source["type"];
	        this.text = source["text"];
	        this.thumb = source["thumb"];
	        this.time = source["time"];
	        this.ts = source["ts"];
	    }
	}
	export class StatusGroupDTO {
	    jid: string;
	    name: string;
	    time: string;
	    mine: boolean;
	    count: number;
	    items: StatusItemDTO[];
	
	    static createFrom(source: any = {}) {
	        return new StatusGroupDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.jid = source["jid"];
	        this.name = source["name"];
	        this.time = source["time"];
	        this.mine = source["mine"];
	        this.count = source["count"];
	        this.items = this.convertValues(source["items"], StatusItemDTO);
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

}

export namespace http {
	
	export class Response {
	    Status: string;
	    StatusCode: number;
	    Proto: string;
	    ProtoMajor: number;
	    ProtoMinor: number;
	    Header: Record<string, Array<string>>;
	    Body: any;
	    ContentLength: number;
	    TransferEncoding: string[];
	    Close: boolean;
	    Uncompressed: boolean;
	    Trailer: Record<string, Array<string>>;
	    Request?: Request;
	    TLS?: tls.ConnectionState;
	
	    static createFrom(source: any = {}) {
	        return new Response(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Status = source["Status"];
	        this.StatusCode = source["StatusCode"];
	        this.Proto = source["Proto"];
	        this.ProtoMajor = source["ProtoMajor"];
	        this.ProtoMinor = source["ProtoMinor"];
	        this.Header = source["Header"];
	        this.Body = source["Body"];
	        this.ContentLength = source["ContentLength"];
	        this.TransferEncoding = source["TransferEncoding"];
	        this.Close = source["Close"];
	        this.Uncompressed = source["Uncompressed"];
	        this.Trailer = source["Trailer"];
	        this.Request = this.convertValues(source["Request"], Request);
	        this.TLS = this.convertValues(source["TLS"], tls.ConnectionState);
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
	export class Request {
	    Method: string;
	    URL?: url.URL;
	    Proto: string;
	    ProtoMajor: number;
	    ProtoMinor: number;
	    Header: Record<string, Array<string>>;
	    Body: any;
	    ContentLength: number;
	    TransferEncoding: string[];
	    Close: boolean;
	    Host: string;
	    Form: Record<string, Array<string>>;
	    PostForm: Record<string, Array<string>>;
	    MultipartForm?: multipart.Form;
	    Trailer: Record<string, Array<string>>;
	    RemoteAddr: string;
	    RequestURI: string;
	    TLS?: tls.ConnectionState;
	    Response?: Response;
	    Pattern: string;
	
	    static createFrom(source: any = {}) {
	        return new Request(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Method = source["Method"];
	        this.URL = this.convertValues(source["URL"], url.URL);
	        this.Proto = source["Proto"];
	        this.ProtoMajor = source["ProtoMajor"];
	        this.ProtoMinor = source["ProtoMinor"];
	        this.Header = source["Header"];
	        this.Body = source["Body"];
	        this.ContentLength = source["ContentLength"];
	        this.TransferEncoding = source["TransferEncoding"];
	        this.Close = source["Close"];
	        this.Host = source["Host"];
	        this.Form = source["Form"];
	        this.PostForm = source["PostForm"];
	        this.MultipartForm = this.convertValues(source["MultipartForm"], multipart.Form);
	        this.Trailer = source["Trailer"];
	        this.RemoteAddr = source["RemoteAddr"];
	        this.RequestURI = source["RequestURI"];
	        this.TLS = this.convertValues(source["TLS"], tls.ConnectionState);
	        this.Response = this.convertValues(source["Response"], Response);
	        this.Pattern = source["Pattern"];
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

}

export namespace multipart {
	
	export class FileHeader {
	    Filename: string;
	    Header: Record<string, Array<string>>;
	    Size: number;
	
	    static createFrom(source: any = {}) {
	        return new FileHeader(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Filename = source["Filename"];
	        this.Header = source["Header"];
	        this.Size = source["Size"];
	    }
	}
	export class Form {
	    Value: Record<string, Array<string>>;
	    File: Record<string, Array<FileHeader>>;
	
	    static createFrom(source: any = {}) {
	        return new Form(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Value = source["Value"];
	        this.File = this.convertValues(source["File"], Array<FileHeader>, true);
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

}

export namespace net {
	
	export class IPNet {
	    IP: number[];
	    Mask: number[];
	
	    static createFrom(source: any = {}) {
	        return new IPNet(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.IP = source["IP"];
	        this.Mask = source["Mask"];
	    }
	}

}

export namespace pkix {
	
	export class AttributeTypeAndValue {
	    Type: number[];
	    Value: any;
	
	    static createFrom(source: any = {}) {
	        return new AttributeTypeAndValue(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Type = source["Type"];
	        this.Value = source["Value"];
	    }
	}
	export class Extension {
	    Id: number[];
	    Critical: boolean;
	    Value: number[];
	
	    static createFrom(source: any = {}) {
	        return new Extension(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Id = source["Id"];
	        this.Critical = source["Critical"];
	        this.Value = source["Value"];
	    }
	}
	export class Name {
	    Country: string[];
	    Organization: string[];
	    OrganizationalUnit: string[];
	    Locality: string[];
	    Province: string[];
	    StreetAddress: string[];
	    PostalCode: string[];
	    SerialNumber: string;
	    CommonName: string;
	    Names: AttributeTypeAndValue[];
	    ExtraNames: AttributeTypeAndValue[];
	
	    static createFrom(source: any = {}) {
	        return new Name(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Country = source["Country"];
	        this.Organization = source["Organization"];
	        this.OrganizationalUnit = source["OrganizationalUnit"];
	        this.Locality = source["Locality"];
	        this.Province = source["Province"];
	        this.StreetAddress = source["StreetAddress"];
	        this.PostalCode = source["PostalCode"];
	        this.SerialNumber = source["SerialNumber"];
	        this.CommonName = source["CommonName"];
	        this.Names = this.convertValues(source["Names"], AttributeTypeAndValue);
	        this.ExtraNames = this.convertValues(source["ExtraNames"], AttributeTypeAndValue);
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

}

export namespace tls {
	
	export class ConnectionState {
	    Version: number;
	    HandshakeComplete: boolean;
	    DidResume: boolean;
	    CipherSuite: number;
	    CurveID: number;
	    NegotiatedProtocol: string;
	    NegotiatedProtocolIsMutual: boolean;
	    ServerName: string;
	    PeerCertificates: x509.Certificate[];
	    VerifiedChains: x509.Certificate[][];
	    SignedCertificateTimestamps: number[][];
	    OCSPResponse: number[];
	    TLSUnique: number[];
	    ECHAccepted: boolean;
	    HelloRetryRequest: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ConnectionState(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Version = source["Version"];
	        this.HandshakeComplete = source["HandshakeComplete"];
	        this.DidResume = source["DidResume"];
	        this.CipherSuite = source["CipherSuite"];
	        this.CurveID = source["CurveID"];
	        this.NegotiatedProtocol = source["NegotiatedProtocol"];
	        this.NegotiatedProtocolIsMutual = source["NegotiatedProtocolIsMutual"];
	        this.ServerName = source["ServerName"];
	        this.PeerCertificates = this.convertValues(source["PeerCertificates"], x509.Certificate);
	        this.VerifiedChains = this.convertValues(source["VerifiedChains"], x509.Certificate);
	        this.SignedCertificateTimestamps = source["SignedCertificateTimestamps"];
	        this.OCSPResponse = source["OCSPResponse"];
	        this.TLSUnique = source["TLSUnique"];
	        this.ECHAccepted = source["ECHAccepted"];
	        this.HelloRetryRequest = source["HelloRetryRequest"];
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

}

export namespace url {
	
	export class Userinfo {
	
	
	    static createFrom(source: any = {}) {
	        return new Userinfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	
	    }
	}
	export class URL {
	    Scheme: string;
	    Opaque: string;
	    // Go type: Userinfo
	    User?: any;
	    Host: string;
	    Path: string;
	    Fragment: string;
	    RawQuery: string;
	    RawPath: string;
	    RawFragment: string;
	    ForceQuery: boolean;
	    OmitHost: boolean;
	
	    static createFrom(source: any = {}) {
	        return new URL(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Scheme = source["Scheme"];
	        this.Opaque = source["Opaque"];
	        this.User = this.convertValues(source["User"], null);
	        this.Host = source["Host"];
	        this.Path = source["Path"];
	        this.Fragment = source["Fragment"];
	        this.RawQuery = source["RawQuery"];
	        this.RawPath = source["RawPath"];
	        this.RawFragment = source["RawFragment"];
	        this.ForceQuery = source["ForceQuery"];
	        this.OmitHost = source["OmitHost"];
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

}

export namespace x509 {
	
	export class PolicyMapping {
	    // Go type: OID
	    IssuerDomainPolicy: any;
	    // Go type: OID
	    SubjectDomainPolicy: any;
	
	    static createFrom(source: any = {}) {
	        return new PolicyMapping(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.IssuerDomainPolicy = this.convertValues(source["IssuerDomainPolicy"], null);
	        this.SubjectDomainPolicy = this.convertValues(source["SubjectDomainPolicy"], null);
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
	export class OID {
	
	
	    static createFrom(source: any = {}) {
	        return new OID(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	
	    }
	}
	export class Certificate {
	    Raw: number[];
	    RawTBSCertificate: number[];
	    RawSubjectPublicKeyInfo: number[];
	    RawSubject: number[];
	    RawIssuer: number[];
	    Signature: number[];
	    SignatureAlgorithm: number;
	    PublicKeyAlgorithm: number;
	    PublicKey: any;
	    Version: number;
	    // Go type: big
	    SerialNumber?: any;
	    Issuer: pkix.Name;
	    Subject: pkix.Name;
	    // Go type: time
	    NotBefore: any;
	    // Go type: time
	    NotAfter: any;
	    KeyUsage: number;
	    Extensions: pkix.Extension[];
	    ExtraExtensions: pkix.Extension[];
	    UnhandledCriticalExtensions: number[][];
	    ExtKeyUsage: number[];
	    UnknownExtKeyUsage: number[][];
	    BasicConstraintsValid: boolean;
	    IsCA: boolean;
	    MaxPathLen: number;
	    MaxPathLenZero: boolean;
	    SubjectKeyId: number[];
	    AuthorityKeyId: number[];
	    OCSPServer: string[];
	    IssuingCertificateURL: string[];
	    DNSNames: string[];
	    EmailAddresses: string[];
	    IPAddresses: number[][];
	    URIs: url.URL[];
	    PermittedDNSDomainsCritical: boolean;
	    PermittedDNSDomains: string[];
	    ExcludedDNSDomains: string[];
	    PermittedIPRanges: net.IPNet[];
	    ExcludedIPRanges: net.IPNet[];
	    PermittedEmailAddresses: string[];
	    ExcludedEmailAddresses: string[];
	    PermittedURIDomains: string[];
	    ExcludedURIDomains: string[];
	    CRLDistributionPoints: string[];
	    PolicyIdentifiers: number[][];
	    Policies: OID[];
	    InhibitAnyPolicy: number;
	    InhibitAnyPolicyZero: boolean;
	    InhibitPolicyMapping: number;
	    InhibitPolicyMappingZero: boolean;
	    RequireExplicitPolicy: number;
	    RequireExplicitPolicyZero: boolean;
	    PolicyMappings: PolicyMapping[];
	
	    static createFrom(source: any = {}) {
	        return new Certificate(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Raw = source["Raw"];
	        this.RawTBSCertificate = source["RawTBSCertificate"];
	        this.RawSubjectPublicKeyInfo = source["RawSubjectPublicKeyInfo"];
	        this.RawSubject = source["RawSubject"];
	        this.RawIssuer = source["RawIssuer"];
	        this.Signature = source["Signature"];
	        this.SignatureAlgorithm = source["SignatureAlgorithm"];
	        this.PublicKeyAlgorithm = source["PublicKeyAlgorithm"];
	        this.PublicKey = source["PublicKey"];
	        this.Version = source["Version"];
	        this.SerialNumber = this.convertValues(source["SerialNumber"], null);
	        this.Issuer = this.convertValues(source["Issuer"], pkix.Name);
	        this.Subject = this.convertValues(source["Subject"], pkix.Name);
	        this.NotBefore = this.convertValues(source["NotBefore"], null);
	        this.NotAfter = this.convertValues(source["NotAfter"], null);
	        this.KeyUsage = source["KeyUsage"];
	        this.Extensions = this.convertValues(source["Extensions"], pkix.Extension);
	        this.ExtraExtensions = this.convertValues(source["ExtraExtensions"], pkix.Extension);
	        this.UnhandledCriticalExtensions = source["UnhandledCriticalExtensions"];
	        this.ExtKeyUsage = source["ExtKeyUsage"];
	        this.UnknownExtKeyUsage = source["UnknownExtKeyUsage"];
	        this.BasicConstraintsValid = source["BasicConstraintsValid"];
	        this.IsCA = source["IsCA"];
	        this.MaxPathLen = source["MaxPathLen"];
	        this.MaxPathLenZero = source["MaxPathLenZero"];
	        this.SubjectKeyId = source["SubjectKeyId"];
	        this.AuthorityKeyId = source["AuthorityKeyId"];
	        this.OCSPServer = source["OCSPServer"];
	        this.IssuingCertificateURL = source["IssuingCertificateURL"];
	        this.DNSNames = source["DNSNames"];
	        this.EmailAddresses = source["EmailAddresses"];
	        this.IPAddresses = source["IPAddresses"];
	        this.URIs = this.convertValues(source["URIs"], url.URL);
	        this.PermittedDNSDomainsCritical = source["PermittedDNSDomainsCritical"];
	        this.PermittedDNSDomains = source["PermittedDNSDomains"];
	        this.ExcludedDNSDomains = source["ExcludedDNSDomains"];
	        this.PermittedIPRanges = this.convertValues(source["PermittedIPRanges"], net.IPNet);
	        this.ExcludedIPRanges = this.convertValues(source["ExcludedIPRanges"], net.IPNet);
	        this.PermittedEmailAddresses = source["PermittedEmailAddresses"];
	        this.ExcludedEmailAddresses = source["ExcludedEmailAddresses"];
	        this.PermittedURIDomains = source["PermittedURIDomains"];
	        this.ExcludedURIDomains = source["ExcludedURIDomains"];
	        this.CRLDistributionPoints = source["CRLDistributionPoints"];
	        this.PolicyIdentifiers = source["PolicyIdentifiers"];
	        this.Policies = this.convertValues(source["Policies"], OID);
	        this.InhibitAnyPolicy = source["InhibitAnyPolicy"];
	        this.InhibitAnyPolicyZero = source["InhibitAnyPolicyZero"];
	        this.InhibitPolicyMapping = source["InhibitPolicyMapping"];
	        this.InhibitPolicyMappingZero = source["InhibitPolicyMappingZero"];
	        this.RequireExplicitPolicy = source["RequireExplicitPolicy"];
	        this.RequireExplicitPolicyZero = source["RequireExplicitPolicyZero"];
	        this.PolicyMappings = this.convertValues(source["PolicyMappings"], PolicyMapping);
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

}

