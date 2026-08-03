import { ChangeDetectionStrategy,Component,OnInit,inject } from '@angular/core';
import { ReactiveFormsModule } from '@angular/forms';
import { MatButtonModule } from '@angular/material/button';
import { MAT_DIALOG_DATA,MatDialogModule,MatDialogRef } from '@angular/material/dialog';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatSelectModule } from '@angular/material/select';
import { MatSlideToggle } from '@angular/material/slide-toggle';
import { FlexDirective } from '@ngbracket/ngx-layout';
import { LayoutGapDirective } from '@ngbracket/ngx-layout/flex';
import { SeedDetailsComponent } from '..';
import { ConfigObject } from '../../../../../shared/models';
import { ConfigDialogData } from '../../../func';
import { SeedMetaComponent } from '../../seed-meta/seed-meta.component';

@Component({
  selector: 'app-entity-dialog',
  templateUrl: './seed-dialog.component.html',
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [
    FlexDirective,
    LayoutGapDirective,
    MatButtonModule,
    MatDialogModule,
    MatFormFieldModule,
    MatSelectModule,
    MatSlideToggle,
    ReactiveFormsModule,
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
}
