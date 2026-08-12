import {
  AfterViewInit,
  ChangeDetectorRef,
  Component,
  ComponentRef,
  Type,
  ViewChild,
  ViewContainerRef,
  inject,
} from '@angular/core';
import {AbstractControl} from '@angular/forms';
import {MatButtonModule} from '@angular/material/button';
import {MAT_DIALOG_DATA, MatDialogModule, MatDialogRef} from '@angular/material/dialog';
import {ConfigObject, Kind} from '../../../../shared/models';
import {AuthService} from '../../../../core';
import {ConfigDialogData} from '../../func';
import {BrowserConfigMultiDialogComponent} from '../browserconfig/browserconfig-multi-dialog/browserconfig-multi-dialog.component';
import {BrowserScriptMultiDialogComponent} from '../browserscript/browserscript-multi-dialog/browserscript-multi-dialog.component';
import {CollectionMultiDialogComponent} from '../collection/collection-multi-dialog/collection-multi-dialog.component';
import {CrawlConfigMultiDialogComponent} from '../crawlconfig/crawlconfig-multi-dialog/crawlconfig-multi-dialog.component';
import {CrawlHostGroupConfigMultiDialogComponent} from '../crawlhostgroupconfig/crawlhostgroupconfig-multi-dialog/crawlhostgroupconfig-multi-dialog.component';
import {CrawlJobMultiDialogComponent} from '../crawljobs/crawljob-multi-dialog/crawljobs-multi-dialog.component';
import {EntityMultiDialogComponent} from '../entity/entity-multi-dialog/entity-multi-dialog.component';
import {PolitenessConfigMultiDialogComponent} from '../politenessconfig/politenessconfig-multi-dialog/politenessconfig-multi-dialog.component';
import {RoleMappingMultiDialogComponent} from '../rolemapping/rolemapping-multi-dialog/rolemapping-multi-dialog.component';
import {ScheduleMultiDialogComponent} from '../schedule/schedule-multi-dialog/schedule-multi-dialog.component';
import {SeedMultiDialogComponent} from '../seed/seed-multi-dialog/seed-multi-dialog.component';

export interface MultiUpdateResult {
  updateTemplate: ConfigObject;
  pathList: string[];
}

export interface MultiUpdateSection {
  form: AbstractControl;
  readonly canUpdate: boolean;
  readonly canRevert: boolean;
  onRevert(): void;
  onDialogClose(): MultiUpdateResult;
}

@Component({
  selector: 'app-multi-update-dialog',
  templateUrl: './multi-update-dialog.component.html',
  styleUrls: ['../mass-update-dialog.scss'],
  imports: [MatButtonModule, MatDialogModule],
  standalone: true,
})
export class MultiUpdateDialogComponent implements AfterViewInit {
  readonly Kind = Kind;
  readonly data = inject<ConfigDialogData>(MAT_DIALOG_DATA);
  private readonly dialogRef = inject<MatDialogRef<MultiUpdateDialogComponent>>(MatDialogRef);
  private readonly cdr = inject(ChangeDetectorRef);
  private readonly authService = inject(AuthService);

  @ViewChild('sectionHost', {read: ViewContainerRef}) sectionHost: ViewContainerRef;
  private sectionRef: ComponentRef<MultiUpdateSection>;

  get section(): MultiUpdateSection | undefined {
    return this.sectionRef?.instance;
  }

  get canEdit(): boolean {
    return this.authService.canUpdate(this.data.configObject.kind);
  }

  ngAfterViewInit(): void {
    this.sectionRef = this.sectionHost.createComponent(sectionByKind(this.data.configObject.kind));
    this.sectionRef.location.nativeElement.classList.add('mass-update-section-host');
    this.cdr.detectChanges();
  }

  onRevert(): void {
    this.section.onRevert();
    this.sectionRef.changeDetectorRef.detectChanges();
    this.cdr.detectChanges();
  }

  onUpdate(): void {
    if (this.section?.canUpdate) {
      this.dialogRef.close(this.section.onDialogClose());
    }
  }
}

function sectionByKind(kind: Kind): Type<MultiUpdateSection> {
  switch (kind) {
    case Kind.CRAWLENTITY: return EntityMultiDialogComponent;
    case Kind.SEED: return SeedMultiDialogComponent;
    case Kind.CRAWLJOB: return CrawlJobMultiDialogComponent;
    case Kind.CRAWLCONFIG: return CrawlConfigMultiDialogComponent;
    case Kind.CRAWLSCHEDULECONFIG: return ScheduleMultiDialogComponent;
    case Kind.BROWSERCONFIG: return BrowserConfigMultiDialogComponent;
    case Kind.BROWSERSCRIPT: return BrowserScriptMultiDialogComponent;
    case Kind.POLITENESSCONFIG: return PolitenessConfigMultiDialogComponent;
    case Kind.CRAWLHOSTGROUPCONFIG: return CrawlHostGroupConfigMultiDialogComponent;
    case Kind.ROLEMAPPING: return RoleMappingMultiDialogComponent;
    case Kind.COLLECTION: return CollectionMultiDialogComponent;
    default: throw new Error(`No multi-update section found for kind: ${kind}`);
  }
}
