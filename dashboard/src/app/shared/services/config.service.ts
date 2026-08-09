import { Injectable, inject } from '@angular/core';
import {from, Observable} from 'rxjs';
import {count, mergeMap} from 'rxjs/operators';
import {create} from '@bufbuild/protobuf';

import {FieldMaskSchema} from '../../../api/commons/v1/resources_pb';
import {
  GetScriptAnnotationsRequestSchema,
  ListRequest,
  ListRequestSchema,
  UpdateRequestSchema
} from '../../../api/config/v1/config_pb';
import {
  Annotation,
  BrowserConfig,
  BrowserScript,
  ConfigObject,
  ConfigRef,
  CrawlConfig,
  CrawlJob,
  Kind,
  Label,
  RoleMapping,
  PolitenessConfig,
  Seed
} from '../models';
import {LoadingService} from '.';
import {ConfigApiService} from '../../core';
import {ConfigQuery, escapeRegex} from '../func';
import {ListRange} from '../models';


@Injectable({
  providedIn: "root"
})
export class ConfigService
  extends LoadingService {
  private configApiService = inject(ConfigApiService);


  // The listRequest last used to fetch data
  private effectiveListRequest: ListRequest;

  get(configRef: ConfigRef): Observable<ConfigObject> {
    return this.load(this.configApiService.get(configRef));
  }

  search(query: ConfigQuery, range: ListRange): Observable<ConfigObject> {
    const listRequest = this.getListRequest(query, range);
    this.effectiveListRequest = listRequest;

    return this.configApiService.list(listRequest);
  }

  count(query: ConfigQuery): Observable<number> {
    const listRequest = this.getListRequest(query, {offset: 0, pageSize: 0});
    return this.configApiService.count(listRequest);
  }

  save(configObject: ConfigObject): Observable<ConfigObject> {
    return this.load(this.configApiService.save(configObject));
  }

  saveMultiple(configObjects: ConfigObject[]): Observable<number> {
    return this.load(from(configObjects).pipe(
      mergeMap(configObject => this.configApiService.save(configObject)),
      count()));
  }

  update(configObject: ConfigObject): Observable<ConfigObject> {
    return this.load(this.configApiService.save(configObject));
  }

  updateWithTemplate(updateTemplate: ConfigObject, paths: string[], ids: string[]): Observable<number> {
    let listRequest: ListRequest;

    if (ids.length > 0) {
      listRequest = create(ListRequestSchema, {
        kind: updateTemplate.kind.valueOf(),
        id: ids
      });
    } else {
      // use previous stored list request as basis
      // but set page size and offset to defaults because we want to update ALL requested objects
      listRequest = this.effectiveListRequest;
      listRequest.pageSize = 0;
      listRequest.offset = 0;
    }

    const updateRequest = create(UpdateRequestSchema, {
      listRequest,
      updateTemplate: ConfigObject.toProto(updateTemplate),
      updateMask: create(FieldMaskSchema, {paths})
    });

    return this.load(this.configApiService.update(updateRequest));
  }

  delete(configObject: ConfigObject): Observable<boolean> {
    return this.load(this.configApiService.delete(configObject));
  }

  deleteMultiple(configObjects: ConfigObject[]): Observable<number> {
    return this.load(from(configObjects).pipe(
      mergeMap((configObject) => this.configApiService.delete(configObject)),
      count(_ => _)
    ));
  }

  move(configObject: ConfigObject, entityRef: ConfigRef): Observable<number> {
    return this.load(this._move(configObject, entityRef));
  }

  moveMultiple(configObjects: ConfigObject[], entityRef: ConfigRef): Observable<number> {
    return this.load(from(configObjects).pipe(
      mergeMap(configObject => this._move(configObject, entityRef)),
      count(updated => !!updated)));
  }

  private _move(configObject: ConfigObject, entityRef: ConfigRef): Observable<number> {
    const refLabel = new Label({key: 'entityRef', value: configObject.seed.entityRef.id});
    const updateTemplate = new ConfigObject({kind: Kind.SEED});
    updateTemplate.meta.labelList = [refLabel];
    updateTemplate.seed.entityRef = entityRef;
    const pathList = ['seed.entityRef', 'meta.label+'];

    return this.updateWithTemplate(updateTemplate, pathList, [configObject.id]);
  }

  private getListRequest(query: ConfigQuery, range: ListRange): ListRequest {
    const listRequest = create(ListRequestSchema, {
      kind: query.kind.valueOf(),
      offset: range.offset,
      pageSize: range.pageSize
    });

    const queryTemplate = new ConfigObject();
    const fieldMask = create(FieldMaskSchema);

    switch (query.kind) {
      case Kind.CRAWLJOB:
        queryTemplate.crawlJob = new CrawlJob();

        if (query.scheduleId) {
          fieldMask.paths.push('crawlJob.scheduleRef');
          queryTemplate.crawlJob.scheduleRef = new ConfigRef({id: query.scheduleId, kind: Kind.CRAWLSCHEDULECONFIG});
        }
        if (query.crawlConfigId) {
          fieldMask.paths.push('crawlJob.crawlConfigRef');
          queryTemplate.crawlJob.crawlConfigRef = new ConfigRef({id: query.crawlConfigId, kind: Kind.CRAWLCONFIG});
        }
        if (query.disabled !== null) {
          fieldMask.paths.push('crawlJob.disabled');
          queryTemplate.crawlJob.disabled = query.disabled;
        }
        break;
      case Kind.CRAWLCONFIG:
        queryTemplate.crawlConfig = new CrawlConfig();

        if (query.collectionId) {
          fieldMask.paths.push('crawlConfig.collectionRef');
          queryTemplate.crawlConfig.collectionRef = new ConfigRef({id: query.collectionId, kind: Kind.COLLECTION});
        }
        if (query.browserConfigId) {
          fieldMask.paths.push('crawlConfig.browserConfigRef');
          queryTemplate.crawlConfig.browserConfigRef = new ConfigRef({
            id: query.browserConfigId,
            kind: Kind.BROWSERCONFIG
          });
        }
        if (query.politenessId) {
          fieldMask.paths.push('crawlConfig.politenessRef');
          queryTemplate.crawlConfig.politenessRef = new ConfigRef({
            id: query.politenessId,
            kind: Kind.POLITENESSCONFIG
          });
        }
        break;
      case Kind.BROWSERCONFIG:
        queryTemplate.browserConfig = new BrowserConfig();

        if (query.scriptIdList.length) {
          fieldMask.paths.push('browserConfig.scriptRef');
          queryTemplate.browserConfig.scriptRefList = query.scriptIdList.map(id => new ConfigRef({
            id,
            kind: Kind.BROWSERSCRIPT
          }));
        }
        break;
      case Kind.BROWSERSCRIPT:
        queryTemplate.browserScript = new BrowserScript();

        if (query.browserScriptType !== null) {
          fieldMask.paths.push('browserScript.browserScriptType');
          queryTemplate.browserScript.browserScriptType = query.browserScriptType;
        }
        break;
      case Kind.POLITENESSCONFIG:
        queryTemplate.politenessConfig = new PolitenessConfig();
        if (query.robotsPolicy !== null) {
          fieldMask.paths.push('politenessConfig.robotsPolicy');
          queryTemplate.politenessConfig.robotsPolicy = query.robotsPolicy;
        }
        break;
      case Kind.SEED:
        queryTemplate.seed = new Seed();

        if (query.entityId) {
          fieldMask.paths.push('seed.entityRef');
          queryTemplate.seed.entityRef = new ConfigRef({id: query.entityId, kind: Kind.CRAWLENTITY});
        }
        if (query.crawlJobIdList.length) {
          fieldMask.paths.push('seed.jobRef');
          queryTemplate.seed.jobRefList = query.crawlJobIdList.map(id => new ConfigRef({id, kind: Kind.CRAWLJOB}));
        }
        if (query.disabled !== null) {
          fieldMask.paths.push('seed.disabled');
          queryTemplate.seed.disabled = query.disabled;
        }
        break;
      case Kind.ROLEMAPPING:
        if (query.term !== null || (query.role !== null && query.role !== undefined)) {
          queryTemplate.roleMapping = new RoleMapping();
          if (query.role !== null && query.role !== undefined) {
            queryTemplate.roleMapping.roleList = [query.role];
            fieldMask.paths.push('roleMapping.role');
          }
          const name = query.term;

          // Search keywords
          // default (no keywords): roleMapping email
          // "group:" roleMapping group
          // "email:" roleMapping email
          if (name !== null) {
            if (name.startsWith('group:')) {
              queryTemplate.roleMapping.group = name.substring(name.indexOf(':') + 1);
              fieldMask.paths.push('roleMapping.group');
            } else {
              let email = name;
              if (name.startsWith('email:')) {
                email = name.substring(name.indexOf(':') + 1);
              }
              queryTemplate.roleMapping.email = email;
              fieldMask.paths.push('roleMapping.email');
            }
          }
        }
        break;
    }

    if (fieldMask.paths.length > 0) {
      listRequest.queryTemplate = ConfigObject.toProto(queryTemplate);
      listRequest.queryMask = fieldMask;
    }

    if (query.direction && query.active) {
      listRequest.orderByPath = 'meta.' + query.active;
      listRequest.orderDescending = query.direction === 'desc';
    }

    if (query.term !== null && query.kind !== Kind.ROLEMAPPING) {
      const labelMarker = 'label:';
      const labelMarkerIndex = query.term.indexOf(labelMarker);
      const label = labelMarkerIndex < 0
        ? ''
        : query.term.slice(labelMarkerIndex + labelMarker.length).trim();
      const name = (labelMarkerIndex < 0
        ? query.term
        : query.term.slice(0, labelMarkerIndex)).trim();

      if (label) {
        listRequest.labelSelector = [label];
      }

      if (name) {
        if (name.startsWith('regex:')) {
          const regex = name.substring(name.indexOf(':') + 1);
          listRequest.nameRegex = regex;
        } else if (name.startsWith('"') && name.endsWith('"')) {
          const exact = '^' + escapeRegex(name.substring(1, name.length - 1)) + '$';
          listRequest.nameRegex = exact;
        } else if (query.kind === Kind.SEED) {
          if (name.startsWith('.')) {
            const subDomainSearch = '^(?:https?://)?.*' + escapeRegex(name) + '/?';
            listRequest.nameRegex = subDomainSearch;
          } else {
            const commonSearch = '^(?:https?://)?(?:w{3}\\.)?' + escapeRegex(name) + '/?';
            listRequest.nameRegex = commonSearch;
          }
        } else {
          listRequest.nameRegex = escapeRegex(name);
        }
      }
    }
    return listRequest;
  }

  getScriptAnnotations(jobId: string, seedId?: string): Observable<Annotation[]> {
    if (jobId) {
      const request = create(GetScriptAnnotationsRequestSchema, {
        job: ConfigRef.toProto(new ConfigRef({kind: Kind.CRAWLJOB, id: jobId}))
      });
      if (seedId) {
        request.seed = ConfigRef.toProto(new ConfigRef({kind: Kind.SEED, id: seedId}));
      }
      return this.configApiService.getScriptAnnotations(request);
    }
    throw new Error('Job ID is required to get script annotations');
  }
}
