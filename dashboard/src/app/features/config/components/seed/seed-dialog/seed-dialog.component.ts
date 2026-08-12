import { ChangeDetectionStrategy,Component,OnInit,inject } from '@angular/core';
import { ReactiveFormsModule } from '@angular/forms';
import { MatButtonModule } from '@angular/material/button';
import { MAT_DIALOG_DATA,MatDialogModule,MatDialogRef } from '@angular/material/dialog';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatSelectModule } from '@angular/material/select';
import { EffectiveScriptAnnotationsComponent, SeedDetailsComponent } from '..';
import { ConfigObject } from '../../../../../shared/models';
import { ConfigDialogData } from '../../../func';
import { SeedMetaComponent } from '../../seed-meta/seed-meta.component';
import {MatSlideToggleModule} from '@angular/material/slide-toggle';
import {MatTooltipModule} from '@angular/material/tooltip';

@Component({
  selector: 'app-entity-dialog',
  templateUrl: './seed-dialog.component.html',
  styleUrls: ['../../config-status-toggle.scss'],
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [
    MatButtonModule,
    MatDialogModule,
    MatFormFieldModule,
    MatSelectModule,
    MatSlideToggleModule,
    MatTooltipModule,
    ReactiveFormsModule,
    EffectiveScriptAnnotationsComponent,
    SeedMetaComponent
  ],
  standalone: true
})
export class SeedDialogComponent extends SeedDetailsComponent implements OnInit {
  private data = inject<ConfigDialogData>(MAT_DIALOG_DATA);
  dialogRef = inject<MatDialogRef<SeedDialogComponent>>(MatDialogRef);

  declare crawlJobs: ConfigObject[];

  constructor() {

    super();
    const data = this.data;

    this.createForm();
    this.crawlJobs = data.options.crawlJobs;
    this.configObject = data.configObject;
  }

  ngOnInit(): void {
    this.updateForm();
  }

  onDialogClose(): ConfigObject | ConfigObject[] {
    return this.isMultipleSeed()
      ? this.prepareSaveMultiple()
      : this.prepareSave();
  }

  onActiveChange(active: boolean): void {
    const disabled = this.form.controls['disabled'];
    disabled.setValue(!active);
    disabled.markAsDirty();
  }
}
